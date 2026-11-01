package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"mongoscale/internal/metrics"
	"mongoscale/internal/utils"
)

const (
	requestIDHeader   = "X-Request-ID"
	virtualUserHeader = "X-Virtual-User"
	slowRequestLimit  = time.Second
)

type Middleware func(http.Handler) http.Handler

func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for index := len(middlewares) - 1; index >= 0; index-- {
		handler = middlewares[index](handler)
	}
	return handler
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(payload []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(payload)
}

func wrapRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{ResponseWriter: w}
}

func Recover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("panic recovered",
						"panic", recovered,
						"path", r.URL.Path,
						"stack", string(debug.Stack()))
					utils.WriteError(w, &utils.AppError{
						Status:  http.StatusInternalServerError,
						Code:    "internal_error",
						Message: "an unexpected error occurred",
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identifier := r.Header.Get(requestIDHeader)
			if identifier == "" {
				buffer := make([]byte, 8)
				_, _ = rand.Read(buffer)
				identifier = hex.EncodeToString(buffer)
			}
			w.Header().Set(requestIDHeader, identifier)
			next.ServeHTTP(w, r)
		})
	}
}

func CORS(allowedOrigin string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headers := w.Header()
			headers.Set("Access-Control-Allow-Origin", allowedOrigin)
			headers.Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
			headers.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Virtual-User, X-Request-ID")
			headers.Set("Access-Control-Expose-Headers", "X-Request-ID")
			headers.Set("Access-Control-Max-Age", "600")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func Timeout(limit time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), limit)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func isObservable(path string) bool {
	return !strings.HasPrefix(path, "/api/v1/system/") && path != "/health" && path != "/metrics"
}

func Metrics(collector *metrics.Collector) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isObservable(r.URL.Path) || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			started := time.Now()
			collector.Begin()
			recorder := wrapRecorder(w)
			defer func() {
				status := recorder.status
				if status == 0 {
					status = http.StatusOK
				}
				collector.End(status, time.Since(started), r.Header.Get(virtualUserHeader))
			}()
			next.ServeHTTP(recorder, r)
		})
	}
}

func AccessLog(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			recorder := wrapRecorder(w)
			next.ServeHTTP(recorder, r)
			elapsed := time.Since(started)
			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}
			attributes := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"durationMs", elapsed.Milliseconds(),
				"requestId", w.Header().Get(requestIDHeader),
			}
			switch {
			case status >= 500:
				logger.Warn("request failed", attributes...)
			case elapsed >= slowRequestLimit:
				logger.Warn("slow request", attributes...)
			default:
				logger.Debug("request served", attributes...)
			}
		})
	}
}
