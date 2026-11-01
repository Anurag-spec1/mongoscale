package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"mongoscale/internal/database"
	"mongoscale/internal/models"
	"mongoscale/internal/utils"
	"mongoscale/pkg/bsonx"
)

type OrderFilter struct {
	UserID *primitive.ObjectID
	Status models.OrderStatus
	Limit  int
	Page   int
	Cursor string
}

type OrderRepository struct {
	db *database.Manager
}

func NewOrderRepository(db *database.Manager) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Insert(ctx context.Context, order *models.Order, tuning models.Tuning) error {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	now := time.Now().UTC()
	order.CreatedAt = now
	order.UpdatedAt = now
	result, err := r.db.Writer("orders", tuning.MajorityWrites).InsertOne(ctx, order)
	if err != nil {
		return err
	}
	order.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Order, error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	var order models.Order
	err := r.db.Plain("orders").FindOne(ctx, bsonx.D(bsonx.E("_id", id))).Decode(&order)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, utils.NotFound("order not found")
	}
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepository) List(ctx context.Context, filter OrderFilter, tuning models.Tuning) (*models.Page[models.Order], error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()

	query := bson.D{}
	if filter.UserID != nil {
		query = append(query, bsonx.E("userId", *filter.UserID))
	}
	if filter.Status != "" {
		query = append(query, bsonx.E("status", filter.Status))
	}
	findOptions := options.Find().SetSort(bsonx.D(bsonx.E("createdAt", -1), bsonx.E("_id", -1)))
	collection := r.db.Reader("orders", tuning.SecondaryReads)

	if tuning.KeysetPagination {
		if filter.Cursor != "" {
			cursorValue, err := utils.DecodeCursor(filter.Cursor)
			if err != nil {
				return nil, err
			}
			clause, err := keysetClause("createdAt", -1, cursorValue, true)
			if err != nil {
				return nil, err
			}
			query = append(query, clause)
		}
		findOptions.SetLimit(int64(filter.Limit + 1))
		cursor, err := collection.Find(ctx, query, findOptions)
		if err != nil {
			return nil, err
		}
		items := make([]models.Order, 0, filter.Limit+1)
		if err := cursor.All(ctx, &items); err != nil {
			return nil, err
		}
		page := &models.Page[models.Order]{Limit: filter.Limit, Pagination: "keyset"}
		if len(items) > filter.Limit {
			items = items[:filter.Limit]
			page.HasMore = true
			last := items[len(items)-1]
			page.NextCursor = dateCursor(last.CreatedAt, last.ID)
		}
		page.Items = items
		return page, nil
	}

	findOptions.SetSkip(offsetFor(filter.Page, filter.Limit)).SetLimit(int64(filter.Limit))
	total, err := collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, err
	}
	cursor, err := collection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, err
	}
	items := make([]models.Order, 0, filter.Limit)
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return &models.Page[models.Order]{
		Items:      items,
		Page:       filter.Page,
		Limit:      filter.Limit,
		Total:      &total,
		HasMore:    offsetFor(filter.Page, filter.Limit)+int64(len(items)) < total,
		Pagination: "offset",
	}, nil
}

func (r *OrderRepository) Summary(ctx context.Context, userID *primitive.ObjectID, tuning models.Tuning) ([]models.OrderSummaryRow, error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	pipeline := mongo.Pipeline{}
	if userID != nil {
		pipeline = append(pipeline, bsonx.D(bsonx.E("$match", bsonx.D(bsonx.E("userId", *userID)))))
	}
	pipeline = append(pipeline,
		bsonx.D(bsonx.E("$group", bsonx.D(
			bsonx.E("_id", "$status"),
			bsonx.E("count", bsonx.D(bsonx.E("$sum", 1))),
			bsonx.E("totalAmount", bsonx.D(bsonx.E("$sum", "$totalAmount"))),
		))),
		bsonx.D(bsonx.E("$sort", bsonx.D(bsonx.E("_id", 1)))),
	)
	cursor, err := r.db.Reader("orders", tuning.SecondaryReads).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	rows := make([]models.OrderSummaryRow, 0, 5)
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *OrderRepository) Transition(
	ctx context.Context,
	order *models.Order,
	next models.OrderStatus,
	payment models.PaymentStatus,
	restock bool,
	products *ProductRepository,
) (*models.Order, error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()

	filter := bsonx.D(bsonx.E("_id", order.ID), bsonx.E("userId", order.UserID), bsonx.E("status", order.Status))
	update := bsonx.D(bsonx.E("$set", bsonx.D(
		bsonx.E("status", next),
		bsonx.E("paymentStatus", payment),
		bsonx.E("updatedAt", time.Now().UTC()),
	)))
	findAndUpdate := options.FindOneAndUpdate().SetReturnDocument(options.After)

	if !restock {
		var updated models.Order
		err := r.db.Writer("orders", true).FindOneAndUpdate(ctx, filter, update, findAndUpdate).Decode(&updated)
		return resolveTransition(&updated, err)
	}

	if r.db.Topology().SupportsTransactions() {
		return r.transitionWithTransaction(ctx, order, filter, update, findAndUpdate, products)
	}

	var updated models.Order
	err := r.db.Writer("orders", true).FindOneAndUpdate(ctx, filter, update, findAndUpdate).Decode(&updated)
	result, resolveErr := resolveTransition(&updated, err)
	if resolveErr != nil {
		return nil, resolveErr
	}
	if err := products.RestoreStock(ctx, order.Items, nil); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *OrderRepository) transitionWithTransaction(
	ctx context.Context,
	order *models.Order,
	filter bson.D,
	update bson.D,
	findAndUpdate *options.FindOneAndUpdateOptions,
	products *ProductRepository,
) (*models.Order, error) {
	session, err := r.db.Client().StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(ctx)

	transactionOptions := options.Transaction().
		SetWriteConcern(writeconcern.New(writeconcern.WMajority())).
		SetReadPreference(readpref.Primary())

	outcome, err := session.WithTransaction(ctx, func(sessionContext mongo.SessionContext) (any, error) {
		var updated models.Order
		decodeErr := r.db.Plain("orders").FindOneAndUpdate(sessionContext, filter, update, findAndUpdate).Decode(&updated)
		if errors.Is(decodeErr, mongo.ErrNoDocuments) {
			return nil, utils.Conflict("order status changed while processing, please retry")
		}
		if decodeErr != nil {
			return nil, decodeErr
		}
		if restoreErr := products.RestoreStock(sessionContext, order.Items, nil); restoreErr != nil {
			return nil, restoreErr
		}
		return &updated, nil
	}, transactionOptions)
	if err != nil {
		return nil, err
	}
	return outcome.(*models.Order), nil
}

func resolveTransition(updated *models.Order, err error) (*models.Order, error) {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, utils.Conflict("order status changed while processing, please retry")
	}
	if err != nil {
		return nil, err
	}
	return updated, nil
}
