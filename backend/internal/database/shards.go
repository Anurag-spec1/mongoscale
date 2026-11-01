package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mongoscale/internal/models"
	"mongoscale/pkg/bsonx"
)

const shardClientServerSelectionTimeout = 3 * time.Second

type shardInfo struct {
	ID   string `bson:"_id"`
	Host string `bson:"host"`
}

func (m *Manager) listShards(ctx context.Context) ([]shardInfo, error) {
	var result struct {
		Shards []shardInfo `bson:"shards"`
	}
	command := bsonx.D(bsonx.E("listShards", 1))
	if err := m.Client().Database("admin").RunCommand(ctx, command).Decode(&result); err != nil {
		return nil, fmt.Errorf("list shards: %w", err)
	}
	return result.Shards, nil
}

func (m *Manager) shardClient(ctx context.Context, host string) (*mongo.Client, error) {
	m.shardMu.Lock()
	defer m.shardMu.Unlock()
	if client, found := m.shardClients[host]; found {
		return client, nil
	}
	uri := "mongodb://" + host
	if replicaSet, members, found := strings.Cut(host, "/"); found {
		uri = "mongodb://" + members + "/?replicaSet=" + replicaSet
	}
	client, err := mongo.Connect(ctx, options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(3).
		SetServerSelectionTimeout(shardClientServerSelectionTimeout))
	if err != nil {
		return nil, fmt.Errorf("connect to shard %s: %w", host, err)
	}
	m.shardClients[host] = client
	return client, nil
}

func (m *Manager) shardMembers(ctx context.Context, host string) []models.MemberStatus {
	client, err := m.shardClient(ctx, host)
	if err != nil {
		m.logger.Debug("shard client unavailable", "host", host, "error", err.Error())
		return nil
	}
	_, members, err := replicaMembers(ctx, client)
	if err != nil {
		m.logger.Debug("shard replica status unavailable", "host", host, "error", err.Error())
		return nil
	}
	return members
}

func (m *Manager) shardDistribution(ctx context.Context) models.ShardDistribution {
	distribution := models.ShardDistribution{Collection: "orders", Shards: []models.ShardStatus{}}
	if !m.Topology().Sharded() {
		return distribution
	}
	shards, err := m.listShards(ctx)
	if err != nil {
		m.logger.Debug("shard listing failed", "error", err.Error())
		return distribution
	}

	var collectionMeta struct {
		Key     bson.D `bson:"key"`
		Dropped bool   `bson:"dropped"`
	}
	metaErr := m.Client().Database("config").Collection("collections").
		FindOne(ctx, bsonx.D(bsonx.E("_id", m.cfg.MongoDatabase+".orders"))).
		Decode(&collectionMeta)
	if metaErr == nil && !collectionMeta.Dropped {
		distribution.Sharded = true
		keyParts := make([]string, 0, len(collectionMeta.Key))
		for _, element := range collectionMeta.Key {
			keyParts = append(keyParts, fmt.Sprintf("%s: %v", element.Key, element.Value))
		}
		distribution.ShardKey = strings.Join(keyParts, ", ")
	}

	type shardStats struct {
		documents int64
		size      int64
	}
	statsByShard := make(map[string]shardStats)
	pipeline := mongo.Pipeline{bsonx.D(bsonx.E("$collStats", bsonx.D(bsonx.E("storageStats", bson.D{}))))}
	cursor, err := m.Plain("orders").Aggregate(ctx, pipeline)
	if err == nil {
		defer cursor.Close(ctx)
		for cursor.Next(ctx) {
			var row struct {
				Shard        string `bson:"shard"`
				StorageStats struct {
					Count int64 `bson:"count"`
					Size  int64 `bson:"size"`
				} `bson:"storageStats"`
			}
			if decodeErr := cursor.Decode(&row); decodeErr == nil {
				current := statsByShard[row.Shard]
				current.documents += row.StorageStats.Count
				current.size += row.StorageStats.Size
				statsByShard[row.Shard] = current
			}
		}
	}

	for _, shard := range shards {
		stats := statsByShard[shard.ID]
		distribution.Shards = append(distribution.Shards, models.ShardStatus{
			Name:      shard.ID,
			Host:      shard.Host,
			Documents: stats.documents,
			SizeBytes: stats.size,
			Members:   m.shardMembers(ctx, shard.Host),
		})
	}
	return distribution
}
