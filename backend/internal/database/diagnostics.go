package database

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"mongoscale/internal/models"
	"mongoscale/pkg/bsonx"
)

var (
	indexNamePattern      = regexp.MustCompile(`"indexName"\s*:\s*"([^"]+)"`)
	collScanPattern       = regexp.MustCompile(`"COLLSCAN"`)
	diagnosticCollections = []string{"users", "products", "orders", "reviews"}
)

type diagnosticSamples struct {
	category  string
	productID primitive.ObjectID
	userID    primitive.ObjectID
}

func (m *Manager) loadSamples(ctx context.Context) *diagnosticSamples {
	m.sampleMu.Lock()
	defer m.sampleMu.Unlock()
	if m.samples != nil {
		return m.samples
	}
	samples := &diagnosticSamples{}
	var product struct {
		Category string `bson:"category"`
	}
	if err := m.Plain("products").FindOne(ctx, bson.D{}).Decode(&product); err != nil {
		return nil
	}
	samples.category = product.Category
	var review struct {
		ProductID primitive.ObjectID `bson:"productId"`
	}
	if err := m.Plain("reviews").FindOne(ctx, bson.D{}).Decode(&review); err == nil {
		samples.productID = review.ProductID
	}
	var order struct {
		UserID primitive.ObjectID `bson:"userId"`
	}
	if err := m.Plain("orders").FindOne(ctx, bson.D{}).Decode(&order); err == nil {
		samples.userID = order.UserID
	}
	m.samples = samples
	return samples
}

func (m *Manager) Diagnose(ctx context.Context, useIndexes bool, includePlans bool) models.MongoStatus {
	status := models.MongoStatus{
		Topology:    m.Topology(),
		Pool:        m.PoolUsage(),
		SlowQueries: m.ops.Slow(),
		OpCounters:  map[string]int64{},
		Indexes:     []models.IndexUsage{},
		QueryPlans:  []models.QueryPlan{},
		Collections: []models.CollectionCount{},
		CollectedAt: time.Now(),
	}
	m.fillServerStatus(ctx, &status)

	if status.Topology.ReplicaSet() {
		name, members, err := replicaMembers(ctx, m.Client())
		if err == nil {
			status.ReplicaSet = &models.ReplicaSetStatus{Name: name, Members: members}
			for _, member := range members {
				if member.State == "PRIMARY" {
					status.ReplicaSet.PrimaryName = member.Name
				}
			}
		} else {
			m.logger.Debug("replica set status unavailable", "error", err.Error())
		}
	}

	status.Shards = m.shardDistribution(ctx)
	status.Indexes = m.indexUsage(ctx)
	if includePlans {
		status.QueryPlans = m.explainSamples(ctx, useIndexes)
	}
	status.Collections = m.collectionCounts(ctx)
	return status
}

func (m *Manager) fillServerStatus(ctx context.Context, status *models.MongoStatus) {
	var raw struct {
		Version     string  `bson:"version"`
		Uptime      float64 `bson:"uptime"`
		Connections struct {
			Current   int64 `bson:"current"`
			Available int64 `bson:"available"`
			Active    int64 `bson:"active"`
		} `bson:"connections"`
		OpCounters map[string]int64 `bson:"opcounters"`
		Mem        struct {
			Resident int64 `bson:"resident"`
			Virtual  int64 `bson:"virtual"`
		} `bson:"mem"`
	}
	command := bsonx.D(bsonx.E("serverStatus", 1))
	if err := m.Client().Database("admin").RunCommand(ctx, command).Decode(&raw); err != nil {
		m.logger.Debug("serverStatus unavailable", "error", err.Error())
		return
	}
	status.Version = raw.Version
	status.UptimeSeconds = raw.Uptime
	status.Connections = models.ConnectionStatus{
		Current:   raw.Connections.Current,
		Available: raw.Connections.Available,
		Active:    raw.Connections.Active,
	}
	if raw.OpCounters != nil {
		status.OpCounters = raw.OpCounters
	}
	status.ResidentMB = raw.Mem.Resident
	status.VirtualMB = raw.Mem.Virtual
}

func replicaMembers(ctx context.Context, client *mongo.Client) (string, []models.MemberStatus, error) {
	var raw struct {
		Set     string `bson:"set"`
		Members []struct {
			Name       string    `bson:"name"`
			StateStr   string    `bson:"stateStr"`
			Health     float64   `bson:"health"`
			Uptime     int64     `bson:"uptime"`
			OptimeDate time.Time `bson:"optimeDate"`
		} `bson:"members"`
	}
	command := bsonx.D(bsonx.E("replSetGetStatus", 1))
	if err := client.Database("admin").RunCommand(ctx, command).Decode(&raw); err != nil {
		return "", nil, fmt.Errorf("replSetGetStatus: %w", err)
	}
	var primaryOptime time.Time
	for _, member := range raw.Members {
		if member.StateStr == "PRIMARY" {
			primaryOptime = member.OptimeDate
		}
	}
	members := make([]models.MemberStatus, 0, len(raw.Members))
	for _, member := range raw.Members {
		lag := 0.0
		if !primaryOptime.IsZero() && !member.OptimeDate.IsZero() && member.StateStr != "PRIMARY" {
			lag = max(primaryOptime.Sub(member.OptimeDate).Seconds(), 0)
		}
		members = append(members, models.MemberStatus{
			Name:          member.Name,
			State:         member.StateStr,
			Healthy:       member.Health >= 1,
			UptimeSeconds: member.Uptime,
			LagSeconds:    lag,
		})
	}
	return raw.Set, members, nil
}

func (m *Manager) indexUsage(ctx context.Context) []models.IndexUsage {
	usage := make([]models.IndexUsage, 0, 16)
	for _, collection := range diagnosticCollections {
		cursor, err := m.Plain(collection).Aggregate(ctx, mongo.Pipeline{bsonx.D(bsonx.E("$indexStats", bson.D{}))})
		if err != nil {
			continue
		}
		positions := make(map[string]int)
		for cursor.Next(ctx) {
			var row struct {
				Name     string `bson:"name"`
				Accesses struct {
					Ops int64 `bson:"ops"`
				} `bson:"accesses"`
			}
			if err := cursor.Decode(&row); err != nil {
				continue
			}
			if position, seen := positions[row.Name]; seen {
				usage[position].Accesses += row.Accesses.Ops
				continue
			}
			positions[row.Name] = len(usage)
			usage = append(usage, models.IndexUsage{Collection: collection, Name: row.Name, Accesses: row.Accesses.Ops})
		}
		_ = cursor.Close(ctx)
	}
	return usage
}

func (m *Manager) collectionCounts(ctx context.Context) []models.CollectionCount {
	counts := make([]models.CollectionCount, 0, len(diagnosticCollections))
	for _, collection := range diagnosticCollections {
		count, err := m.Plain(collection).EstimatedDocumentCount(ctx)
		if err != nil {
			continue
		}
		counts = append(counts, models.CollectionCount{Name: collection, Count: count})
	}
	return counts
}

type explainCase struct {
	name       string
	collection string
	filter     string
	command    bson.D
}

func (m *Manager) explainSamples(ctx context.Context, useIndexes bool) []models.QueryPlan {
	samples := m.loadSamples(ctx)
	if samples == nil {
		return []models.QueryPlan{}
	}
	searchFilter := bsonx.D(bsonx.E("name", bsonx.D(bsonx.E("$regex", "pro"), bsonx.E("$options", "i"))))
	searchLabel := "name =~ /pro/i"
	if useIndexes {
		searchFilter = bsonx.D(bsonx.E("$text", bsonx.D(bsonx.E("$search", "pro"))))
		searchLabel = "$text: pro"
	}
	cases := []explainCase{
		{
			name:       "Category browse sorted by price",
			collection: "products",
			filter:     "category = " + samples.category + ", sort price",
			command: bsonx.D(
				bsonx.E("find", "products"),
				bsonx.E("filter", bsonx.D(bsonx.E("category", samples.category))),
				bsonx.E("sort", bsonx.D(bsonx.E("price", 1))),
				bsonx.E("limit", 20),
			),
		},
		{
			name:       "Product search",
			collection: "products",
			filter:     searchLabel,
			command: bsonx.D(
				bsonx.E("find", "products"),
				bsonx.E("filter", searchFilter),
				bsonx.E("limit", 20),
			),
		},
		{
			name:       "Reviews for a product",
			collection: "reviews",
			filter:     "productId = ?, sort createdAt desc",
			command: bsonx.D(
				bsonx.E("find", "reviews"),
				bsonx.E("filter", bsonx.D(bsonx.E("productId", samples.productID))),
				bsonx.E("sort", bsonx.D(bsonx.E("createdAt", -1))),
				bsonx.E("limit", 20),
			),
		},
		{
			name:       "Order history for a user",
			collection: "orders",
			filter:     "userId = ?, sort createdAt desc",
			command: bsonx.D(
				bsonx.E("find", "orders"),
				bsonx.E("filter", bsonx.D(bsonx.E("userId", samples.userID))),
				bsonx.E("sort", bsonx.D(bsonx.E("createdAt", -1))),
				bsonx.E("limit", 20),
			),
		},
	}

	plans := make([]models.QueryPlan, 0, len(cases))
	for _, explainRequest := range cases {
		plan, err := m.runExplain(ctx, explainRequest)
		if err != nil {
			m.logger.Debug("explain failed", "query", explainRequest.name, "error", err.Error())
			continue
		}
		plans = append(plans, plan)
	}
	return plans
}

func (m *Manager) runExplain(ctx context.Context, request explainCase) (models.QueryPlan, error) {
	command := bsonx.D(bsonx.E("explain", request.command), bsonx.E("verbosity", "executionStats"))
	var raw struct {
		QueryPlanner struct {
			WinningPlan bson.Raw `bson:"winningPlan"`
		} `bson:"queryPlanner"`
		ExecutionStats struct {
			Returned     int64 `bson:"nReturned"`
			Millis       int64 `bson:"executionTimeMillis"`
			KeysExamined int64 `bson:"totalKeysExamined"`
			DocsExamined int64 `bson:"totalDocsExamined"`
		} `bson:"executionStats"`
	}
	if err := m.current.Load().database.RunCommand(ctx, command).Decode(&raw); err != nil {
		return models.QueryPlan{}, err
	}
	plan := raw.QueryPlanner.WinningPlan.String()
	stage := "IXSCAN"
	if collScanPattern.MatchString(plan) {
		stage = "COLLSCAN"
	}
	indexName := ""
	if match := indexNamePattern.FindStringSubmatch(plan); len(match) == 2 {
		indexName = match[1]
	}
	return models.QueryPlan{
		Name:            request.name,
		Collection:      request.collection,
		Filter:          request.filter,
		Stage:           stage,
		IndexName:       indexName,
		DocsExamined:    raw.ExecutionStats.DocsExamined,
		KeysExamined:    raw.ExecutionStats.KeysExamined,
		Returned:        raw.ExecutionStats.Returned,
		ExecutionMillis: raw.ExecutionStats.Millis,
	}, nil
}

func (m *Manager) StepDown(ctx context.Context, shard string) error {
	var target *mongo.Client
	switch {
	case m.Topology().ReplicaSet():
		target = m.Client()
	case m.Topology().Sharded():
		shards, err := m.listShards(ctx)
		if err != nil {
			return err
		}
		for _, candidate := range shards {
			if candidate.ID == shard {
				client, clientErr := m.shardClient(ctx, candidate.Host)
				if clientErr != nil {
					return clientErr
				}
				target = client
			}
		}
		if target == nil {
			return fmt.Errorf("shard %q was not found", shard)
		}
	default:
		return ErrUnsupportedTopology
	}
	command := bsonx.D(
		bsonx.E("replSetStepDown", 30),
		bsonx.E("secondaryCatchUpPeriodSecs", 10),
		bsonx.E("force", true),
	)
	err := target.Database("admin").RunCommand(ctx, command).Err()
	if err != nil && !mongo.IsNetworkError(err) {
		return fmt.Errorf("step down primary: %w", err)
	}
	return nil
}
