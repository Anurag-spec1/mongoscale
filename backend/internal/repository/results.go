package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mongoscale/internal/database"
	"mongoscale/internal/models"
	"mongoscale/pkg/bsonx"
)

type ResultRepository struct {
	db *database.Manager
}

func NewResultRepository(db *database.Manager) *ResultRepository {
	return &ResultRepository{db: db}
}

func (r *ResultRepository) Insert(ctx context.Context, result *models.LoadTestResult) error {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	inserted, err := r.db.Writer("loadtest_results", false).InsertOne(ctx, result)
	if err != nil {
		return err
	}
	result.ID = inserted.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ResultRepository) Recent(ctx context.Context, limit int) ([]models.LoadTestResult, error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	cursor, err := r.db.Plain("loadtest_results").Find(ctx, bson.D{},
		options.Find().SetSort(bsonx.D(bsonx.E("finishedAt", -1))).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	results := make([]models.LoadTestResult, 0, limit)
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}
