package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mongoscale/internal/database"
	"mongoscale/internal/models"
	"mongoscale/internal/utils"
	"mongoscale/pkg/bsonx"
)

type ReviewRepository struct {
	db *database.Manager
}

func NewReviewRepository(db *database.Manager) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) Create(ctx context.Context, review *models.Review) error {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	now := time.Now().UTC()
	review.CreatedAt = now
	review.UpdatedAt = now
	result, err := r.db.Writer("reviews", true).InsertOne(ctx, review)
	if err != nil {
		return err
	}
	review.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ReviewRepository) ListByProduct(
	ctx context.Context,
	productID primitive.ObjectID,
	limit, page int,
	cursorToken string,
	tuning models.Tuning,
) (*models.Page[models.Review], error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()

	query := bsonx.D(bsonx.E("productId", productID))
	findOptions := options.Find().SetSort(bsonx.D(bsonx.E("createdAt", -1), bsonx.E("_id", -1)))
	collection := r.db.Reader("reviews", tuning.SecondaryReads)

	if tuning.KeysetPagination {
		if cursorToken != "" {
			cursorValue, err := utils.DecodeCursor(cursorToken)
			if err != nil {
				return nil, err
			}
			clause, err := keysetClause("createdAt", -1, cursorValue, true)
			if err != nil {
				return nil, err
			}
			query = append(query, clause)
		}
		findOptions.SetLimit(int64(limit + 1))
		cursor, err := collection.Find(ctx, query, findOptions)
		if err != nil {
			return nil, err
		}
		items := make([]models.Review, 0, limit+1)
		if err := cursor.All(ctx, &items); err != nil {
			return nil, err
		}
		result := &models.Page[models.Review]{Limit: limit, Pagination: "keyset"}
		if len(items) > limit {
			items = items[:limit]
			result.HasMore = true
			last := items[len(items)-1]
			result.NextCursor = dateCursor(last.CreatedAt, last.ID)
		}
		result.Items = items
		return result, nil
	}

	findOptions.SetSkip(offsetFor(page, limit)).SetLimit(int64(limit))
	total, err := collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, err
	}
	cursor, err := collection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, err
	}
	items := make([]models.Review, 0, limit)
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return &models.Page[models.Review]{
		Items:      items,
		Page:       page,
		Limit:      limit,
		Total:      &total,
		HasMore:    offsetFor(page, limit)+int64(len(items)) < total,
		Pagination: "offset",
	}, nil
}

func (r *ReviewRepository) Summary(ctx context.Context, productID primitive.ObjectID, tuning models.Tuning) (*models.ReviewSummary, error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	pipeline := mongo.Pipeline{
		bsonx.D(bsonx.E("$match", bsonx.D(bsonx.E("productId", productID)))),
		bsonx.D(bsonx.E("$group", bsonx.D(
			bsonx.E("_id", "$rating"),
			bsonx.E("count", bsonx.D(bsonx.E("$sum", 1))),
		))),
	}
	cursor, err := r.db.Reader("reviews", tuning.SecondaryReads).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Rating int   `bson:"_id"`
		Count  int64 `bson:"count"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	summary := &models.ReviewSummary{
		ProductID:    productID.Hex(),
		Distribution: map[int]int64{1: 0, 2: 0, 3: 0, 4: 0, 5: 0},
	}
	var weighted int64
	for _, row := range rows {
		summary.Distribution[row.Rating] = row.Count
		summary.TotalReviews += row.Count
		weighted += int64(row.Rating) * row.Count
	}
	if summary.TotalReviews > 0 {
		summary.AverageRating = float64(weighted) / float64(summary.TotalReviews)
	}
	return summary, nil
}
