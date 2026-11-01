package repository

import (
	"context"
	"errors"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mongoscale/internal/database"
	"mongoscale/internal/models"
	"mongoscale/internal/utils"
	"mongoscale/pkg/bsonx"
)

type ProductFilter struct {
	Category string
	Query    string
	MinPrice *float64
	MaxPrice *float64
	Sort     string
	Limit    int
	Page     int
	Cursor   string
}

type ProductRepository struct {
	db *database.Manager
}

func NewProductRepository(db *database.Manager) *ProductRepository {
	return &ProductRepository{db: db}
}

func sortSpecification(sortKey string) (field string, direction int, numeric bool) {
	switch sortKey {
	case "price_desc":
		return "price", -1, true
	case "rating":
		return "rating", -1, true
	case "newest":
		return "createdAt", -1, false
	}
	return "price", 1, true
}

func (r *ProductRepository) List(ctx context.Context, filter ProductFilter, tuning models.Tuning) (*models.Page[models.Product], error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()

	query := bson.D{}
	if filter.Category != "" {
		query = append(query, bsonx.E("category", filter.Category))
	}
	if filter.Query != "" {
		if tuning.Indexes {
			query = append(query, bsonx.E("$text", bsonx.D(bsonx.E("$search", filter.Query))))
		} else {
			query = append(query, bsonx.E("name", bsonx.D(
				bsonx.E("$regex", regexp.QuoteMeta(filter.Query)),
				bsonx.E("$options", "i"),
			)))
		}
	}
	if filter.MinPrice != nil || filter.MaxPrice != nil {
		priceRange := bson.D{}
		if filter.MinPrice != nil {
			priceRange = append(priceRange, bsonx.E("$gte", *filter.MinPrice))
		}
		if filter.MaxPrice != nil {
			priceRange = append(priceRange, bsonx.E("$lte", *filter.MaxPrice))
		}
		query = append(query, bsonx.E("price", priceRange))
	}

	field, direction, numeric := sortSpecification(filter.Sort)
	findOptions := options.Find().SetSort(bsonx.D(bsonx.E(field, direction), bsonx.E("_id", direction)))
	if tuning.Projections {
		findOptions.SetProjection(bsonx.D(bsonx.E("description", 0), bsonx.E("lastReservation", 0)))
	}
	collection := r.db.Reader("products", tuning.SecondaryReads)

	if tuning.KeysetPagination {
		return r.listKeyset(ctx, collection, query, findOptions, filter, field, direction, numeric)
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
	items := make([]models.Product, 0, filter.Limit)
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return &models.Page[models.Product]{
		Items:      items,
		Page:       filter.Page,
		Limit:      filter.Limit,
		Total:      &total,
		HasMore:    offsetFor(filter.Page, filter.Limit)+int64(len(items)) < total,
		Pagination: "offset",
	}, nil
}

func (r *ProductRepository) listKeyset(
	ctx context.Context,
	collection *mongo.Collection,
	query bson.D,
	findOptions *options.FindOptions,
	filter ProductFilter,
	field string,
	direction int,
	numeric bool,
) (*models.Page[models.Product], error) {
	if filter.Cursor != "" {
		cursorValue, err := utils.DecodeCursor(filter.Cursor)
		if err != nil {
			return nil, err
		}
		clause, err := keysetClause(field, direction, cursorValue, !numeric)
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
	items := make([]models.Product, 0, filter.Limit+1)
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	page := &models.Page[models.Product]{Limit: filter.Limit, Pagination: "keyset"}
	if len(items) > filter.Limit {
		items = items[:filter.Limit]
		page.HasMore = true
		last := items[len(items)-1]
		switch field {
		case "price":
			page.NextCursor = numberCursor(last.Price, last.ID)
		case "rating":
			page.NextCursor = numberCursor(last.Rating, last.ID)
		default:
			page.NextCursor = dateCursor(last.CreatedAt, last.ID)
		}
	}
	page.Items = items
	return page, nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id primitive.ObjectID, tuning models.Tuning) (*models.Product, error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	var product models.Product
	err := r.db.Reader("products", tuning.SecondaryReads).FindOne(ctx, bsonx.D(bsonx.E("_id", id))).Decode(&product)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, utils.NotFound("product not found")
	}
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepository) Exists(ctx context.Context, id primitive.ObjectID) (bool, error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	count, err := r.db.Plain("products").CountDocuments(ctx, bsonx.D(bsonx.E("_id", id)), options.Count().SetLimit(1))
	return count > 0, err
}

func (r *ProductRepository) FindByIDs(ctx context.Context, ids []primitive.ObjectID) (map[primitive.ObjectID]models.Product, error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	projection := bsonx.D(bsonx.E("name", 1), bsonx.E("price", 1), bsonx.E("stock", 1))
	cursor, err := r.db.Plain("products").Find(ctx,
		bsonx.D(bsonx.E("_id", bsonx.D(bsonx.E("$in", ids)))),
		options.Find().SetProjection(projection))
	if err != nil {
		return nil, err
	}
	var found []models.Product
	if err := cursor.All(ctx, &found); err != nil {
		return nil, err
	}
	byID := make(map[primitive.ObjectID]models.Product, len(found))
	for _, product := range found {
		byID[product.ID] = product
	}
	return byID, nil
}

func (r *ProductRepository) AllRefs(ctx context.Context) ([]models.ProductRef, error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	cursor, err := r.db.Plain("products").Find(ctx, bson.D{},
		options.Find().SetProjection(bsonx.D(bsonx.E("category", 1), bsonx.E("price", 1))))
	if err != nil {
		return nil, err
	}
	var refs []models.ProductRef
	if err := cursor.All(ctx, &refs); err != nil {
		return nil, err
	}
	return refs, nil
}

func (r *ProductRepository) Categories(ctx context.Context, tuning models.Tuning) ([]string, error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	values, err := r.db.Reader("products", tuning.SecondaryReads).Distinct(ctx, "category", bson.D{})
	if err != nil {
		return nil, err
	}
	categories := make([]string, 0, len(values))
	for _, value := range values {
		if name, ok := value.(string); ok {
			categories = append(categories, name)
		}
	}
	return categories, nil
}

func (r *ProductRepository) Create(ctx context.Context, product *models.Product) error {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	now := time.Now().UTC()
	product.CreatedAt = now
	product.UpdatedAt = now
	result, err := r.db.Writer("products", true).InsertOne(ctx, product)
	if err != nil {
		return err
	}
	product.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ProductRepository) AdjustStock(ctx context.Context, id primitive.ObjectID, delta int) (*models.Product, error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	filter := bsonx.D(bsonx.E("_id", id))
	if delta < 0 {
		filter = append(filter, bsonx.E("stock", bsonx.D(bsonx.E("$gte", -delta))))
	}
	update := bsonx.D(
		bsonx.E("$inc", bsonx.D(bsonx.E("stock", delta))),
		bsonx.E("$set", bsonx.D(bsonx.E("updatedAt", time.Now().UTC()))),
	)
	var product models.Product
	err := r.db.Writer("products", true).
		FindOneAndUpdate(ctx, filter, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).
		Decode(&product)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, utils.Conflict("product does not exist or does not have enough stock")
	}
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepository) ReserveStock(ctx context.Context, token string, items []models.OrderItem) (int64, error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	operations := make([]mongo.WriteModel, 0, len(items))
	now := time.Now().UTC()
	for _, item := range items {
		operations = append(operations, mongo.NewUpdateOneModel().
			SetFilter(bsonx.D(bsonx.E("_id", item.ProductID), bsonx.E("stock", bsonx.D(bsonx.E("$gte", item.Quantity))))).
			SetUpdate(bsonx.D(
				bsonx.E("$inc", bsonx.D(bsonx.E("stock", -item.Quantity))),
				bsonx.E("$set", bsonx.D(bsonx.E("lastReservation", token), bsonx.E("updatedAt", now))),
			)))
	}
	result, err := r.db.Writer("products", true).BulkWrite(ctx, operations, options.BulkWrite().SetOrdered(false))
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

func (r *ProductRepository) ReleaseReservation(ctx context.Context, token string, items []models.OrderItem) error {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	cursor, err := r.db.Plain("products").Find(ctx,
		bsonx.D(bsonx.E("lastReservation", token)),
		options.Find().SetProjection(bsonx.D(bsonx.E("_id", 1))))
	if err != nil {
		return err
	}
	var reserved []models.Product
	if err := cursor.All(ctx, &reserved); err != nil {
		return err
	}
	reservedIDs := make(map[primitive.ObjectID]bool, len(reserved))
	for _, product := range reserved {
		reservedIDs[product.ID] = true
	}
	return r.RestoreStock(ctx, items, func(item models.OrderItem) bool { return reservedIDs[item.ProductID] })
}

func (r *ProductRepository) RestoreStock(ctx context.Context, items []models.OrderItem, include func(models.OrderItem) bool) error {
	operations := make([]mongo.WriteModel, 0, len(items))
	for _, item := range items {
		if include != nil && !include(item) {
			continue
		}
		operations = append(operations, mongo.NewUpdateOneModel().
			SetFilter(bsonx.D(bsonx.E("_id", item.ProductID))).
			SetUpdate(bsonx.D(bsonx.E("$inc", bsonx.D(bsonx.E("stock", item.Quantity))))))
	}
	if len(operations) == 0 {
		return nil
	}
	_, err := r.db.Writer("products", true).BulkWrite(ctx, operations, options.BulkWrite().SetOrdered(false))
	return err
}

func (r *ProductRepository) ApplyRating(ctx context.Context, id primitive.ObjectID, rating int) error {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	pipeline := mongo.Pipeline{
		bsonx.D(bsonx.E("$set", bsonx.D(
			bsonx.E("rating", bsonx.D(bsonx.E("$divide", bsonx.A(
				bsonx.D(bsonx.E("$add", bsonx.A(
					bsonx.D(bsonx.E("$multiply", bsonx.A("$rating", "$ratingCount"))),
					rating,
				))),
				bsonx.D(bsonx.E("$add", bsonx.A("$ratingCount", 1))),
			)))),
			bsonx.E("ratingCount", bsonx.D(bsonx.E("$add", bsonx.A("$ratingCount", 1)))),
			bsonx.E("updatedAt", time.Now().UTC()),
		))),
	}
	_, err := r.db.Writer("products", true).UpdateOne(ctx, bsonx.D(bsonx.E("_id", id)), pipeline)
	return err
}
