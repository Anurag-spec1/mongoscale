package loadtest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"mongoscale/internal/models"
)

const (
	maxIdentities    = 2000
	controlInterval  = 200 * time.Millisecond
	requestTimeout   = 10 * time.Second
	samplingTimeout  = 15 * time.Second
	phaseRampingUp   = "ramping-up"
	phaseHolding     = "holding"
	phaseRampingDown = "ramping-down"
	phaseIdle        = "idle"
	phaseStarting    = "starting"
	phaseStopped     = "stopped"
	phaseFinished    = "finished"
)

var ErrAlreadyRunning = errors.New("a load test is already running")

type Source interface {
	SampleUsers(ctx context.Context, limit int) ([]models.UserRef, error)
	ProductRefs(ctx context.Context) ([]models.ProductRef, error)
	IssueToken(userID string, role models.Role) (string, error)
}

type Plan struct {
	Users    int
	RampUp   time.Duration
	Hold     time.Duration
	RampDown time.Duration
	ThinkMin time.Duration
	ThinkMax time.Duration
}

func (p Plan) Total() time.Duration {
	return p.RampUp + p.Hold + p.RampDown
}

type Report struct {
	Plan       Plan
	Mode       models.Mode
	PeakUsers  int
	Requests   int64
	Failures   int64
	Throttled  int64
	P50        float64
	P95        float64
	P99        float64
	StartedAt  time.Time
	FinishedAt time.Time
	Completed  bool
}

type identity struct {
	userID string
	token  string
}

type runState struct {
	plan       Plan
	mode       models.Mode
	identities []identity
	products   []models.ProductRef
	started    time.Time
	histogram  *histogram
	requests   atomic.Int64
	failures   atomic.Int64
	throttled  atomic.Int64
	active     atomic.Int64
	peak       atomic.Int64
}

type Runner struct {
	baseURL string
	source  Source
	logger  *slog.Logger
	client  *http.Client

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	status  models.LoadTestStatus
	current *runState
}

func NewRunner(baseURL string, source Source, logger *slog.Logger) *Runner {
	transport := &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns:        0,
		MaxIdleConnsPerHost: 20000,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  true,
	}
	return &Runner{
		baseURL: baseURL,
		source:  source,
		logger:  logger,
		client:  &http.Client{Transport: transport, Timeout: requestTimeout},
		status:  models.LoadTestStatus{Phase: phaseIdle},
	}
}

func (r *Runner) Start(plan Plan, mode models.Mode, onFinish func(Report)) error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return ErrAlreadyRunning
	}
	r.running = true
	r.status = models.LoadTestStatus{
		Running:      true,
		Phase:        phaseStarting,
		Mode:         mode,
		TargetUsers:  plan.Users,
		TotalSeconds: plan.Total().Seconds(),
		StartedAt:    time.Now().UTC(),
	}
	r.mu.Unlock()

	state, err := r.prepare(plan, mode)
	if err != nil {
		r.mu.Lock()
		r.running = false
		r.status.Running = false
		r.status.Phase = phaseIdle
		r.status.Message = err.Error()
		r.mu.Unlock()
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.mu.Lock()
	r.cancel = cancel
	r.current = state
	r.mu.Unlock()

	go r.execute(ctx, state, onFinish)
	return nil
}

func (r *Runner) prepare(plan Plan, mode models.Mode) (*runState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), samplingTimeout)
	defer cancel()

	users, err := r.source.SampleUsers(ctx, min(plan.Users, maxIdentities))
	if err != nil {
		return nil, fmt.Errorf("sample users: %w", err)
	}
	if len(users) == 0 {
		return nil, errors.New("no customer accounts found; run the seed command first")
	}
	products, err := r.source.ProductRefs(ctx)
	if err != nil {
		return nil, fmt.Errorf("load products: %w", err)
	}
	if len(products) == 0 {
		return nil, errors.New("no products found; run the seed command first")
	}
	identities := make([]identity, 0, len(users))
	for _, user := range users {
		token, tokenErr := r.source.IssueToken(user.ID.Hex(), user.Role)
		if tokenErr != nil {
			return nil, tokenErr
		}
		identities = append(identities, identity{userID: user.ID.Hex(), token: token})
	}
	return &runState{
		plan:       plan,
		mode:       mode,
		identities: identities,
		products:   products,
		started:    time.Now(),
		histogram:  &histogram{},
	}, nil
}

func (r *Runner) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running && r.cancel != nil {
		r.cancel()
	}
}

func (r *Runner) Status() models.LoadTestStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}

func (r *Runner) Running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

func (r *Runner) execute(ctx context.Context, state *runState, onFinish func(Report)) {
	var workers sync.WaitGroup
	var cancels []context.CancelFunc
	nextID := 0

	scale := func(desired int) {
		for len(cancels) < desired {
			workerCtx, cancelWorker := context.WithCancel(ctx)
			cancels = append(cancels, cancelWorker)
			nextID++
			workers.Add(1)
			state.active.Add(1)
			go r.virtualUser(workerCtx, nextID, state, &workers)
		}
		for len(cancels) > desired {
			last := len(cancels) - 1
			cancels[last]()
			cancels = cancels[:last]
		}
		if active := state.active.Load(); active > state.peak.Load() {
			state.peak.Store(active)
		}
	}

	ticker := time.NewTicker(controlInterval)
	defer ticker.Stop()
	completed := true
	for {
		elapsed := time.Since(state.started)
		phase, desired := desiredLoad(state.plan, elapsed)
		scale(desired)
		r.publish(state, phase, elapsed)
		if elapsed >= state.plan.Total() {
			break
		}
		select {
		case <-ctx.Done():
			completed = false
		case <-ticker.C:
		}
		if !completed {
			break
		}
	}

	scale(0)
	workers.Wait()
	finished := time.Now()

	report := Report{
		Plan:       state.plan,
		Mode:       state.mode,
		PeakUsers:  int(state.peak.Load()),
		Requests:   state.requests.Load(),
		Failures:   state.failures.Load(),
		Throttled:  state.throttled.Load(),
		P50:        state.histogram.percentile(0.50),
		P95:        state.histogram.percentile(0.95),
		P99:        state.histogram.percentile(0.99),
		StartedAt:  state.started,
		FinishedAt: finished,
		Completed:  completed,
	}

	r.mu.Lock()
	r.running = false
	r.cancel = nil
	r.status.Running = false
	r.status.ActiveUsers = 0
	r.status.Requests = report.Requests
	r.status.Failures = report.Failures
	r.status.Throttled = report.Throttled
	r.status.ElapsedSeconds = finished.Sub(state.started).Seconds()
	if completed {
		r.status.Phase = phaseFinished
		r.status.ProgressPercent = 100
	} else {
		r.status.Phase = phaseStopped
	}
	r.mu.Unlock()

	if onFinish != nil && report.Requests > 0 {
		onFinish(report)
	}
}

func desiredLoad(plan Plan, elapsed time.Duration) (string, int) {
	switch {
	case elapsed < plan.RampUp:
		fraction := float64(elapsed) / float64(plan.RampUp)
		return phaseRampingUp, int(math.Ceil(float64(plan.Users) * fraction))
	case elapsed < plan.RampUp+plan.Hold:
		return phaseHolding, plan.Users
	case elapsed < plan.Total():
		fraction := float64(elapsed-plan.RampUp-plan.Hold) / float64(plan.RampDown)
		return phaseRampingDown, int(math.Ceil(float64(plan.Users) * (1 - fraction)))
	}
	return phaseRampingDown, 0
}

func (r *Runner) publish(state *runState, phase string, elapsed time.Duration) {
	total := state.plan.Total().Seconds()
	progress := 0.0
	if total > 0 {
		progress = math.Min(elapsed.Seconds()/total*100, 100)
	}
	r.mu.Lock()
	r.status.Phase = phase
	r.status.ActiveUsers = int(state.active.Load())
	r.status.PeakUsers = int(state.peak.Load())
	r.status.Requests = state.requests.Load()
	r.status.Failures = state.failures.Load()
	r.status.Throttled = state.throttled.Load()
	r.status.ElapsedSeconds = elapsed.Seconds()
	r.status.ProgressPercent = progress
	r.mu.Unlock()
}
