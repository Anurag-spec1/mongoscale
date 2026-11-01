package metrics

import (
	"context"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"mongoscale/internal/models"
)

const (
	historyCapacity   = 600
	activeUserWindow  = 5 * time.Second
	activeUserExpiry  = 30 * time.Second
	smoothingWindow   = 3
	criticalErrorRate = 10.0
	criticalP95       = 2000.0
	degradedErrorRate = 2.0
	degradedP95       = 500.0
)

type Point struct {
	Timestamp   int64   `json:"timestamp"`
	RPS         float64 `json:"rps"`
	P50         float64 `json:"p50"`
	P95         float64 `json:"p95"`
	P99         float64 `json:"p99"`
	ErrorRate   float64 `json:"errorRate"`
	Errors      int     `json:"errors"`
	Throttled   int     `json:"throttled"`
	ActiveUsers int     `json:"activeUsers"`
	Inflight    int64   `json:"inflight"`
	MongoReads  float64 `json:"mongoReads"`
	MongoWrites float64 `json:"mongoWrites"`
	SlowQueries int64   `json:"slowQueries"`
	PoolInUse   int64   `json:"poolInUse"`
	PoolWaiting int64   `json:"poolWaiting"`
	CPUPercent  float64 `json:"cpuPercent"`
	MemoryMB    float64 `json:"memoryMB"`
	Goroutines  int     `json:"goroutines"`
}

type Totals struct {
	Requests  int64 `json:"requests"`
	Errors    int64 `json:"errors"`
	Throttled int64 `json:"throttled"`
}

type Snapshot struct {
	Status  string  `json:"status"`
	Current Point   `json:"current"`
	Totals  Totals  `json:"totals"`
	History []Point `json:"history"`
}

type PoolSource func() models.PoolUsage

type Collector struct {
	inflight       atomic.Int64
	totalRequests  atomic.Int64
	totalErrors    atomic.Int64
	totalThrottled atomic.Int64
	virtualUsers   sync.Map
	ops            *MongoOps
	pool           PoolSource

	mu        sync.Mutex
	latencies []float64
	errors    int
	throttled int
	history   []Point

	lastTick     time.Time
	lastReads    int64
	lastWrites   int64
	lastCPUTime  time.Duration
	totalSamples atomic.Int64
}

func NewCollector(ops *MongoOps, pool PoolSource) *Collector {
	return &Collector{
		ops:       ops,
		pool:      pool,
		latencies: make([]float64, 0, 4096),
		history:   make([]Point, 0, historyCapacity),
		lastTick:  time.Now(),
	}
}

func (c *Collector) Begin() {
	c.inflight.Add(1)
}

func (c *Collector) End(status int, duration time.Duration, virtualUser string) {
	c.inflight.Add(-1)
	c.totalRequests.Add(1)
	if virtualUser != "" {
		c.virtualUsers.Store(virtualUser, time.Now().UnixNano())
	}
	milliseconds := float64(duration.Microseconds()) / 1000
	c.mu.Lock()
	c.latencies = append(c.latencies, milliseconds)
	switch {
	case status == 429:
		c.throttled++
	case status >= 500:
		c.errors++
	}
	c.mu.Unlock()
	switch {
	case status == 429:
		c.totalThrottled.Add(1)
	case status >= 500:
		c.totalErrors.Add(1)
	}
}

func (c *Collector) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	c.lastCPUTime = processCPUTime()
	c.lastReads = c.ops.Reads()
	c.lastWrites = c.ops.Writes()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			c.tick(now)
		}
	}
}

func (c *Collector) tick(now time.Time) {
	c.mu.Lock()
	latencies := c.latencies
	errorCount := c.errors
	throttledCount := c.throttled
	c.latencies = make([]float64, 0, max(len(latencies), 1024))
	c.errors = 0
	c.throttled = 0
	c.mu.Unlock()

	elapsed := now.Sub(c.lastTick).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}
	c.lastTick = now

	sort.Float64s(latencies)
	requests := len(latencies)
	point := Point{
		Timestamp:   now.UnixMilli(),
		RPS:         float64(requests) / elapsed,
		P50:         percentile(latencies, 0.50),
		P95:         percentile(latencies, 0.95),
		P99:         percentile(latencies, 0.99),
		Errors:      errorCount,
		Throttled:   throttledCount,
		ActiveUsers: c.activeUsers(now),
		Inflight:    c.inflight.Load(),
		SlowQueries: c.ops.Slow(),
		Goroutines:  runtime.NumGoroutine(),
	}
	if requests > 0 {
		point.ErrorRate = float64(errorCount) / float64(requests) * 100
	}

	reads := c.ops.Reads()
	writes := c.ops.Writes()
	point.MongoReads = float64(reads-c.lastReads) / elapsed
	point.MongoWrites = float64(writes-c.lastWrites) / elapsed
	c.lastReads = reads
	c.lastWrites = writes

	cpuTime := processCPUTime()
	point.CPUPercent = float64(cpuTime-c.lastCPUTime) / (elapsed * float64(time.Second) * float64(runtime.NumCPU())) * 100
	c.lastCPUTime = cpuTime

	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	point.MemoryMB = float64(memory.Alloc) / 1024 / 1024

	usage := c.pool()
	point.PoolInUse = usage.InUse
	point.PoolWaiting = usage.Waiting

	c.mu.Lock()
	if len(c.history) >= historyCapacity {
		copy(c.history, c.history[1:])
		c.history = c.history[:historyCapacity-1]
	}
	c.history = append(c.history, point)
	c.mu.Unlock()
}

func (c *Collector) activeUsers(now time.Time) int {
	active := 0
	windowStart := now.Add(-activeUserWindow).UnixNano()
	expiry := now.Add(-activeUserExpiry).UnixNano()
	c.virtualUsers.Range(func(key, value any) bool {
		seen := value.(int64)
		if seen >= windowStart {
			active++
		} else if seen < expiry {
			c.virtualUsers.Delete(key)
		}
		return true
	})
	return active
}

func (c *Collector) Snapshot(historyLimit int) Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if historyLimit <= 0 || historyLimit > len(c.history) {
		historyLimit = len(c.history)
	}
	history := make([]Point, historyLimit)
	copy(history, c.history[len(c.history)-historyLimit:])
	snapshot := Snapshot{
		Status:  classify(c.history),
		History: history,
		Totals: Totals{
			Requests:  c.totalRequests.Load(),
			Errors:    c.totalErrors.Load(),
			Throttled: c.totalThrottled.Load(),
		},
	}
	if len(c.history) > 0 {
		snapshot.Current = c.history[len(c.history)-1]
	}
	return snapshot
}

func (c *Collector) ActiveUsers() int {
	return c.activeUsers(time.Now())
}

func classify(history []Point) string {
	if len(history) == 0 {
		return "idle"
	}
	start := max(len(history)-smoothingWindow, 0)
	recent := history[start:]
	var rps, errorRate, p95 float64
	for _, point := range recent {
		rps += point.RPS
		errorRate += point.ErrorRate
		p95 += point.P95
	}
	count := float64(len(recent))
	rps /= count
	errorRate /= count
	p95 /= count
	switch {
	case rps < 1 && recent[len(recent)-1].Inflight == 0:
		return "idle"
	case errorRate >= criticalErrorRate || p95 >= criticalP95:
		return "critical"
	case errorRate >= degradedErrorRate || p95 >= degradedP95:
		return "degraded"
	}
	return "healthy"
}

func percentile(sorted []float64, fraction float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)-1) * fraction)
	return sorted[index]
}
