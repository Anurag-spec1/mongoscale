package database

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mongoscale/pkg/bsonx"
)

var optionalIndexCollections = []string{"products", "orders", "reviews"}

func coreIndexes() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys:    bsonx.D(bsonx.E("email", 1)),
			Options: options.Index().SetName("email_1").SetUnique(true),
		},
	}
}

func optionalIndexes() map[string][]mongo.IndexModel {
	return map[string][]mongo.IndexModel{
		"products": {
			{
				Keys:    bsonx.D(bsonx.E("category", 1), bsonx.E("price", 1)),
				Options: options.Index().SetName("category_1_price_1"),
			},
			{
				Keys:    bsonx.D(bsonx.E("name", "text")),
				Options: options.Index().SetName("name_text"),
			},
			{
				Keys:    bsonx.D(bsonx.E("rating", -1)),
				Options: options.Index().SetName("rating_-1"),
			},
		},
		"orders": {
			{
				Keys:    bsonx.D(bsonx.E("userId", 1), bsonx.E("createdAt", -1)),
				Options: options.Index().SetName("userId_1_createdAt_-1"),
			},
			{
				Keys:    bsonx.D(bsonx.E("status", 1), bsonx.E("createdAt", -1)),
				Options: options.Index().SetName("status_1_createdAt_-1"),
			},
			{
				Keys:    bsonx.D(bsonx.E("userId", "hashed")),
				Options: options.Index().SetName("userId_hashed"),
			},
		},
		"reviews": {
			{
				Keys:    bsonx.D(bsonx.E("productId", 1), bsonx.E("createdAt", -1)),
				Options: options.Index().SetName("productId_1_createdAt_-1"),
			},
			{
				Keys:    bsonx.D(bsonx.E("productId", 1), bsonx.E("rating", -1)),
				Options: options.Index().SetName("productId_1_rating_-1"),
			},
		},
	}
}

func (m *Manager) EnsureCoreIndexes(ctx context.Context) error {
	if _, err := m.Plain("users").Indexes().CreateMany(ctx, coreIndexes()); err != nil {
		return fmt.Errorf("create users indexes: %w", err)
	}
	return nil
}

func (m *Manager) EnsureOptionalIndexes(ctx context.Context) error {
	for collection, models := range optionalIndexes() {
		if _, err := m.Plain(collection).Indexes().CreateMany(ctx, models); err != nil {
			return fmt.Errorf("create %s indexes: %w", collection, err)
		}
	}
	return nil
}

func (m *Manager) DropOptionalIndexes(ctx context.Context) error {
	for _, collection := range optionalIndexCollections {
		indexes := m.Plain(collection).Indexes()
		specifications, err := indexes.ListSpecifications(ctx)
		if err != nil {
			return fmt.Errorf("list %s indexes: %w", collection, err)
		}
		for _, specification := range specifications {
			if specification.Name == "_id_" {
				continue
			}
			if _, err := indexes.DropOne(ctx, specification.Name); err != nil {
				m.logger.Warn("index could not be dropped",
					"collection", collection,
					"index", specification.Name,
					"error", err.Error())
			}
		}
	}
	return nil
}

func (m *Manager) ShardOrders(ctx context.Context) error {
	if !m.Topology().Sharded() {
		return ErrUnsupportedTopology
	}
	hashedIndex := mongo.IndexModel{
		Keys:    bsonx.D(bsonx.E("userId", "hashed")),
		Options: options.Index().SetName("userId_hashed"),
	}
	if _, err := m.Plain("orders").Indexes().CreateOne(ctx, hashedIndex); err != nil {
		return fmt.Errorf("create hashed shard key index: %w", err)
	}
	admin := m.Client().Database("admin")
	_ = admin.RunCommand(ctx, bsonx.D(bsonx.E("enableSharding", m.cfg.MongoDatabase))).Err()
	command := bsonx.D(
		bsonx.E("shardCollection", m.cfg.MongoDatabase+".orders"),
		bsonx.E("key", bsonx.D(bsonx.E("userId", "hashed"))),
	)
	if err := admin.RunCommand(ctx, command).Err(); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "already sharded") {
			return nil
		}
		return fmt.Errorf("shard orders collection: %w", err)
	}
	return nil
}
