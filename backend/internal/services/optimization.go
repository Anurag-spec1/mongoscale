package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"mongoscale/internal/cache"
	"mongoscale/internal/database"
	"mongoscale/internal/models"
)

type OptimizationService struct {
	db      *database.Manager
	cache   *cache.Cache
	logger  *slog.Logger
	current atomic.Pointer[models.Tuning]
	mu      sync.Mutex
}

func NewOptimizationService(db *database.Manager, cacheClient *cache.Cache, logger *slog.Logger) *OptimizationService {
	return &OptimizationService{db: db, cache: cacheClient, logger: logger}
}

func (s *OptimizationService) Tuning() models.Tuning {
	if tuning := s.current.Load(); tuning != nil {
		return *tuning
	}
	fallback, _ := models.TuningFor(models.ModeUnoptimized)
	return fallback
}

func (s *OptimizationService) Apply(ctx context.Context, mode models.Mode) (*models.ModeChange, error) {
	tuning, ok := models.TuningFor(mode)
	if !ok {
		return nil, fmt.Errorf("unknown mode %q", mode)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	started := time.Now()
	var warnings []string

	if !s.db.Connected() || s.db.Profile().MaxPoolSize != tuning.MaxPoolSize {
		if err := s.db.Connect(ctx, database.Profile{MaxPoolSize: tuning.MaxPoolSize, MinPoolSize: tuning.MinPoolSize}); err != nil {
			return nil, err
		}
	}

	if tuning.Indexes {
		if err := s.db.EnsureCoreIndexes(ctx); err != nil {
			return nil, err
		}
		if err := s.db.EnsureOptionalIndexes(ctx); err != nil {
			return nil, err
		}
		s.current.Store(&tuning)
	} else {
		s.current.Store(&tuning)
		if err := s.db.DropOptionalIndexes(ctx); err != nil {
			return nil, err
		}
	}

	topology := s.db.Topology()
	if tuning.Sharding {
		if topology.Sharded() {
			if err := s.db.ShardOrders(ctx); err != nil {
				warnings = append(warnings, "orders could not be sharded: "+err.Error())
			}
		} else {
			warnings = append(warnings, "no sharded cluster detected; start docker-compose.sharded.yml to see hashed userId sharding")
		}
	}
	if tuning.SecondaryReads && !topology.ReplicaSet() && !topology.Sharded() {
		warnings = append(warnings, "no replica set detected; reads are served by the primary. Start docker-compose.replica.yml to see secondaryPreferred reads")
	}
	if tuning.Cache && !s.cache.Enabled() {
		warnings = append(warnings, "redis is not reachable; caching is inactive")
	}

	s.cache.Flush(ctx)

	s.logger.Info("optimization mode applied",
		"mode", string(mode),
		"topology", topology.Kind,
		"warnings", len(warnings),
		"durationMs", time.Since(started).Milliseconds())

	return &models.ModeChange{
		Mode:       mode,
		Tuning:     tuning,
		Topology:   topology,
		Warnings:   warnings,
		AppliedAt:  time.Now().UTC(),
		DurationMs: time.Since(started).Milliseconds(),
	}, nil
}
