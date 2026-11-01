package middleware

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"mongoscale/internal/utils"
)

const (
	limiterIdleExpiry     = 2 * time.Minute
	limiterCleanupPeriod  = time.Minute
	realIPHeader          = "X-Real-IP"
	retryAfterHeaderValue = "1"
)

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	limit   rate.Limit
	burst   int
	mu      sync.Mutex
	clients map[string]*clientLimiter
}

func NewRateLimiter(requestsPerSecond float64, burst int) *RateLimiter {
	return &RateLimiter{
		limit:   rate.Limit(requestsPerSecond),
		burst:   burst,
		clients: make(map[string]*clientLimiter),
	}
}

func (l *RateLimiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	client, found := l.clients[key]
	if !found {
		client = &clientLimiter{limiter: rate.NewLimiter(l.limit, l.burst)}
		l.clients[key] = client
	}
	client.lastSeen = now
	l.mu.Unlock()
	return client.limiter.Allow()
}

func (l *RateLimiter) Run(ctx context.Context) {
	ticker := time.NewTicker(limiterCleanupPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			l.mu.Lock()
			for key, client := range l.clients {
				if now.Sub(client.lastSeen) > limiterIdleExpiry {
					delete(l.clients, key)
				}
			}
			l.mu.Unlock()
		}
	}
}

func clientKey(r *http.Request) string {
	if identity, ok := IdentityFrom(r.Context()); ok {
		return "user:" + identity.UserID.Hex()
	}
	if forwarded := r.Header.Get(realIPHeader); forwarded != "" {
		return "ip:" + forwarded
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "ip:" + r.RemoteAddr
	}
	return "ip:" + host
}

func RateLimit(enabled func() bool, limiter *RateLimiter, applies func(*http.Request) bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if enabled() && (applies == nil || applies(r)) && !limiter.Allow(clientKey(r)) {
				w.Header().Set("Retry-After", retryAfterHeaderValue)
				utils.WriteError(w, utils.TooManyRequests("too many requests, slow down"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
