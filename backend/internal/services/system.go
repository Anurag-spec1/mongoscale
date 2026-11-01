package services

import (
	"context"
	"io"
	"sync"
	"time"

	"mongoscale/internal/cache"
	"mongoscale/internal/database"
	"mongoscale/internal/metrics"
	"mongoscale/internal/models"
)

const (
	statusCacheTTL = 3 * time.Second
	plansCacheTTL  = 15 * time.Second
	healthTimeout  = 2 * time.Second
)

type MetricsResponse struct {
	Snapshot      metrics.Snapshot `json:"snapshot"`
	Mode          models.Mode      `json:"mode"`
	Tuning        models.Tuning    `json:"tuning"`
	Topology      models.Topology  `json:"topology"`
	Pool          models.PoolUsage `json:"pool"`
	Cache         cache.Stats      `json:"cache"`
	SlowQueries   int64            `json:"slowQueries"`
	MongoFailures int64            `json:"mongoFailures"`
	UptimeSeconds float64          `json:"uptimeSeconds"`
}

type SystemService struct {
	db           *database.Manager
	cache        *cache.Cache
	collector    *metrics.Collector
	ops          *metrics.MongoOps
	optimization *OptimizationService
	started      time.Time

	mu        sync.Mutex
	status    *models.MongoStatus
	statusAt  time.Time
	plans     []models.QueryPlan
	plansAt   time.Time
	plansMode models.Mode
}

func NewSystemService(
	db *database.Manager,
	cacheClient *cache.Cache,
	collector *metrics.Collector,
	ops *metrics.MongoOps,
	optimization *OptimizationService,
) *SystemService {
	return &SystemService{
		db:           db,
		cache:        cacheClient,
		collector:    collector,
		ops:          ops,
		optimization: optimization,
		started:      time.Now(),
	}
}

func (s *SystemService) Health(ctx context.Context) (models.HealthReport, bool) {
	ctx, cancel := context.WithTimeout(ctx, healthTimeout)
	defer cancel()
	report := models.HealthReport{
		Status:        "ok",
		Mongo:         "ok",
		Redis:         "disabled",
		Mode:          s.optimization.Tuning().Mode,
		UptimeSeconds: time.Since(s.started).Seconds(),
	}
	healthy := true
	if !s.db.Connected() {
		report.Mongo = "disconnected"
		report.Status = "unavailable"
		return report, false
	}
	report.Topology = s.db.Topology().Kind
	if err := s.db.Ping(ctx); err != nil {
		report.Mongo = "unreachable"
		report.Status = "unavailable"
		healthy = false
	}
	if s.cache.Enabled() {
		report.Redis = "ok"
		if err := s.cache.Ping(ctx); err != nil {
			report.Redis = "unreachable"
			if healthy {
				report.Status = "degraded"
			}
		}
	}
	return report, healthy
}

func (s *SystemService) Metrics(historyLimit int) MetricsResponse {
	tuning := s.optimization.Tuning()
	return MetricsResponse{
		Snapshot:      s.collector.Snapshot(historyLimit),
		Mode:          tuning.Mode,
		Tuning:        tuning,
		Topology:      s.db.Topology(),
		Pool:          s.db.PoolUsage(),
		Cache:         s.cache.Stats(),
		SlowQueries:   s.ops.Slow(),
		MongoFailures: s.ops.Failed(),
		UptimeSeconds: time.Since(s.started).Seconds(),
	}
}

func (s *SystemService) MongoStatus(ctx context.Context) models.MongoStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	tuning := s.optimization.Tuning()
	if s.status != nil && now.Sub(s.statusAt) < statusCacheTTL && s.plansMode == tuning.Mode {
		return *s.status
	}
	refreshPlans := s.plans == nil || now.Sub(s.plansAt) >= plansCacheTTL || s.plansMode != tuning.Mode
	status := s.db.Diagnose(ctx, tuning.Indexes, refreshPlans)
	if refreshPlans {
		s.plans = status.QueryPlans
		s.plansAt = now
		s.plansMode = tuning.Mode
	} else {
		status.QueryPlans = s.plans
	}
	s.status = &status
	s.statusAt = now
	return status
}

func (s *SystemService) InvalidateStatus() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = nil
	s.plans = nil
}

func (s *SystemService) StepDown(ctx context.Context, shard string) error {
	err := s.db.StepDown(ctx, shard)
	s.InvalidateStatus()
	return err
}

func (s *SystemService) WritePrometheus(w io.Writer) {
	s.collector.WritePrometheus(w, s.db.PoolUsage, string(s.optimization.Tuning().Mode))
}
