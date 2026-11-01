package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"mongoscale/internal/config"
	"mongoscale/internal/metrics"
	"mongoscale/internal/models"
	"mongoscale/pkg/bsonx"
)

const (
	connectTimeout          = 5 * time.Second
	serverSelectionTimeout  = 5 * time.Second
	maxConnectionIdleTime   = 60 * time.Second
	retiredClientGracePause = 15 * time.Second
)

var ErrUnsupportedTopology = errors.New("operation is not supported by the current MongoDB topology")

type Profile struct {
	MaxPoolSize uint64
	MinPoolSize uint64
}

type connection struct {
	client   *mongo.Client
	database *mongo.Database
	profile  Profile
	pool     *metrics.PoolStats
	topology models.Topology
}

type Manager struct {
	cfg    *config.Config
	logger *slog.Logger
	ops    *metrics.MongoOps

	current  atomic.Pointer[connection]
	switchMu sync.Mutex

	shardMu      sync.Mutex
	shardClients map[string]*mongo.Client

	sampleMu sync.Mutex
	samples  *diagnosticSamples
}

func NewManager(cfg *config.Config, logger *slog.Logger, ops *metrics.MongoOps) *Manager {
	return &Manager{
		cfg:          cfg,
		logger:       logger,
		ops:          ops,
		shardClients: make(map[string]*mongo.Client),
	}
}

func (m *Manager) Connect(ctx context.Context, profile Profile) error {
	m.switchMu.Lock()
	defer m.switchMu.Unlock()

	pool := metrics.NewPoolStats(profile.MaxPoolSize)
	commandMonitor := &event.CommandMonitor{
		Succeeded: func(_ context.Context, finished *event.CommandSucceededEvent) {
			m.ops.Record(finished.CommandName, finished.Duration, false)
		},
		Failed: func(_ context.Context, finished *event.CommandFailedEvent) {
			m.ops.Record(finished.CommandName, finished.Duration, true)
		},
	}
	clientOptions := options.Client().
		ApplyURI(m.cfg.MongoURI).
		SetAppName("mongoscale-api").
		SetMaxPoolSize(profile.MaxPoolSize).
		SetMinPoolSize(profile.MinPoolSize).
		SetConnectTimeout(connectTimeout).
		SetServerSelectionTimeout(serverSelectionTimeout).
		SetMaxConnIdleTime(maxConnectionIdleTime).
		SetPoolMonitor(&event.PoolMonitor{Event: pool.Handle}).
		SetMonitor(commandMonitor)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("connect to mongodb: %w", err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return fmt.Errorf("ping mongodb primary: %w", err)
	}
	topology, err := detectTopology(ctx, client)
	if err != nil {
		_ = client.Disconnect(context.Background())
		return err
	}

	next := &connection{
		client:   client,
		database: client.Database(m.cfg.MongoDatabase),
		profile:  profile,
		pool:     pool,
		topology: topology,
	}
	previous := m.current.Swap(next)
	if previous != nil {
		time.AfterFunc(retiredClientGracePause, func() {
			disconnectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = previous.client.Disconnect(disconnectCtx)
		})
	}
	m.logger.Info("mongodb connection established",
		slog.String("topology", topology.Kind),
		slog.Uint64("maxPoolSize", profile.MaxPoolSize),
		slog.Uint64("minPoolSize", profile.MinPoolSize))
	return nil
}

func detectTopology(ctx context.Context, client *mongo.Client) (models.Topology, error) {
	var hello struct {
		Message string `bson:"msg"`
		SetName string `bson:"setName"`
	}
	command := bsonx.D(bsonx.E("hello", 1))
	if err := client.Database("admin").RunCommand(ctx, command).Decode(&hello); err != nil {
		return models.Topology{}, fmt.Errorf("detect topology: %w", err)
	}
	switch {
	case hello.Message == "isdbgrid":
		return models.Topology{Kind: "sharded"}, nil
	case hello.SetName != "":
		return models.Topology{Kind: "replica-set", SetName: hello.SetName}, nil
	}
	return models.Topology{Kind: "standalone"}, nil
}

func (m *Manager) Client() *mongo.Client {
	return m.current.Load().client
}

func (m *Manager) Topology() models.Topology {
	return m.current.Load().topology
}

func (m *Manager) Profile() Profile {
	return m.current.Load().profile
}

func (m *Manager) Connected() bool {
	return m.current.Load() != nil
}

func (m *Manager) DatabaseName() string {
	return m.cfg.MongoDatabase
}

func (m *Manager) PoolUsage() models.PoolUsage {
	connection := m.current.Load()
	if connection == nil {
		return models.PoolUsage{}
	}
	return connection.pool.Usage()
}

func (m *Manager) Plain(name string) *mongo.Collection {
	return m.current.Load().database.Collection(name)
}

func (m *Manager) Reader(name string, secondaryPreferred bool) *mongo.Collection {
	database := m.current.Load().database
	if secondaryPreferred {
		return database.Collection(name, options.Collection().SetReadPreference(readpref.SecondaryPreferred()))
	}
	return database.Collection(name)
}

func (m *Manager) Writer(name string, majority bool) *mongo.Collection {
	database := m.current.Load().database
	if majority {
		return database.Collection(name, options.Collection().SetWriteConcern(writeconcern.New(writeconcern.WMajority())))
	}
	return database.Collection(name, options.Collection().SetWriteConcern(writeconcern.New(writeconcern.W(1))))
}

func (m *Manager) OpContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, m.cfg.MongoOpTimeout)
}

func (m *Manager) Ping(ctx context.Context) error {
	connection := m.current.Load()
	if connection == nil {
		return errors.New("mongodb is not connected")
	}
	return connection.client.Ping(ctx, readpref.Primary())
}

func (m *Manager) Close(ctx context.Context) {
	if connection := m.current.Swap(nil); connection != nil {
		_ = connection.client.Disconnect(ctx)
	}
	m.shardMu.Lock()
	defer m.shardMu.Unlock()
	for key, client := range m.shardClients {
		_ = client.Disconnect(ctx)
		delete(m.shardClients, key)
	}
}
