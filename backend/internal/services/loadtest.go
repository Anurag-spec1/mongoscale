package services

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"mongoscale/internal/database"
	"mongoscale/internal/loadtest"
	"mongoscale/internal/models"
	"mongoscale/internal/repository"
	"mongoscale/internal/utils"
)

const (
	maxLoadTestUsers   = 20000
	maxLoadTestSeconds = 900
	defaultRampUp      = 20
	defaultHold        = 40
	defaultRampDown    = 10
	defaultThinkMin    = 800
	defaultThinkMax    = 2500
)

type loadTestSource struct {
	users    *repository.UserRepository
	products *repository.ProductRepository
	auth     *AuthService
}

func (s loadTestSource) SampleUsers(ctx context.Context, limit int) ([]models.UserRef, error) {
	return s.users.SampleCustomers(ctx, limit)
}

func (s loadTestSource) ProductRefs(ctx context.Context) ([]models.ProductRef, error) {
	return s.products.AllRefs(ctx)
}

func (s loadTestSource) IssueToken(userID string, role models.Role) (string, error) {
	return s.auth.IssueToken(userID, role)
}

type LoadTestService struct {
	runner       *loadtest.Runner
	results      *repository.ResultRepository
	optimization *OptimizationService
	db           *database.Manager
	logger       *slog.Logger
}

func NewLoadTestService(
	baseURL string,
	users *repository.UserRepository,
	products *repository.ProductRepository,
	auth *AuthService,
	results *repository.ResultRepository,
	optimization *OptimizationService,
	db *database.Manager,
	logger *slog.Logger,
) *LoadTestService {
	source := loadTestSource{users: users, products: products, auth: auth}
	return &LoadTestService{
		runner:       loadtest.NewRunner(baseURL, source, logger),
		results:      results,
		optimization: optimization,
		db:           db,
		logger:       logger,
	}
}

func (s *LoadTestService) Start(request models.LoadTestRequest) (models.LoadTestStatus, error) {
	plan, err := buildPlan(request)
	if err != nil {
		return models.LoadTestStatus{}, err
	}
	mode := s.optimization.Tuning().Mode
	topology := s.db.Topology().Kind
	err = s.runner.Start(plan, mode, func(report loadtest.Report) {
		s.record(report, topology)
	})
	if err != nil {
		if errors.Is(err, loadtest.ErrAlreadyRunning) {
			return models.LoadTestStatus{}, utils.Conflict(err.Error())
		}
		return models.LoadTestStatus{}, utils.BadRequest(err.Error())
	}
	return s.runner.Status(), nil
}

func buildPlan(request models.LoadTestRequest) (loadtest.Plan, error) {
	if request.Users < 1 || request.Users > maxLoadTestUsers {
		return loadtest.Plan{}, utils.BadRequest("users must be between 1 and 20000")
	}
	rampUp := valueOr(request.RampUpSeconds, defaultRampUp)
	hold := valueOr(request.HoldSeconds, defaultHold)
	rampDown := valueOr(request.RampDownSeconds, defaultRampDown)
	thinkMin := valueOr(request.ThinkTimeMinMs, defaultThinkMin)
	thinkMax := valueOr(request.ThinkTimeMaxMs, defaultThinkMax)
	if rampUp+hold+rampDown > maxLoadTestSeconds || rampUp < 0 || hold < 0 || rampDown < 0 {
		return loadtest.Plan{}, utils.BadRequest("the total duration must be at most 900 seconds")
	}
	if rampUp+hold+rampDown == 0 {
		return loadtest.Plan{}, utils.BadRequest("the total duration must be greater than zero")
	}
	if thinkMin < 0 || thinkMax < thinkMin || thinkMax > 60000 {
		return loadtest.Plan{}, utils.BadRequest("think time must satisfy 0 <= min <= max <= 60000 ms")
	}
	return loadtest.Plan{
		Users:    request.Users,
		RampUp:   time.Duration(rampUp) * time.Second,
		Hold:     time.Duration(hold) * time.Second,
		RampDown: time.Duration(rampDown) * time.Second,
		ThinkMin: time.Duration(thinkMin) * time.Millisecond,
		ThinkMax: time.Duration(thinkMax) * time.Millisecond,
	}, nil
}

func valueOr(value, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

func (s *LoadTestService) record(report loadtest.Report, topology string) {
	if !report.Completed {
		return
	}
	duration := report.FinishedAt.Sub(report.StartedAt).Seconds()
	result := &models.LoadTestResult{
		Mode:            report.Mode,
		Topology:        topology,
		TargetUsers:     report.Plan.Users,
		PeakUsers:       report.PeakUsers,
		DurationSeconds: duration,
		TotalRequests:   report.Requests,
		FailedRequests:  report.Failures,
		Throttled:       report.Throttled,
		P50:             report.P50,
		P95:             report.P95,
		P99:             report.P99,
		StartedAt:       report.StartedAt.UTC(),
		FinishedAt:      report.FinishedAt.UTC(),
	}
	if report.Requests > 0 {
		result.ErrorRate = float64(report.Failures) / float64(report.Requests) * 100
	}
	if duration > 0 {
		result.RPS = float64(report.Requests) / duration
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.results.Insert(ctx, result); err != nil {
		s.logger.Error("load test result could not be stored", "error", err.Error())
	}
}

func (s *LoadTestService) Stop() {
	s.runner.Stop()
}

func (s *LoadTestService) Status() models.LoadTestStatus {
	return s.runner.Status()
}

func (s *LoadTestService) Results(ctx context.Context) ([]models.LoadTestResult, error) {
	return s.results.Recent(ctx, 50)
}
