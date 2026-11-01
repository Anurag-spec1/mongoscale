package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mongoscale/internal/cache"
	"mongoscale/internal/config"
	"mongoscale/internal/database"
	"mongoscale/internal/handlers"
	"mongoscale/internal/metrics"
	"mongoscale/internal/middleware"
	"mongoscale/internal/models"
	"mongoscale/internal/repository"
	"mongoscale/internal/services"
)

const (
	startupTimeout       = 90 * time.Second
	startupRetryInterval = 3 * time.Second
	readHeaderTimeout    = 5 * time.Second
	readTimeout          = 15 * time.Second
	idleTimeout          = 120 * time.Second
	maxHeaderBytes       = 1 << 16
	writeTimeoutHeadroom = 5 * time.Second
	healthcheckTimeout   = 3 * time.Second
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe the local health endpoint and exit")
	flag.Parse()
	if *healthcheck {
		os.Exit(probeHealth())
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "mongoscale api stopped:", err)
		os.Exit(1)
	}
}

func probeHealth() int {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}
	client := &http.Client{Timeout: healthcheckTimeout}
	response, err := client.Get("http://127.0.0.1:" + port + "/health")
	if err != nil {
		return 1
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func newLogger(level string) *slog.Logger {
	var parsed slog.Level
	if err := parsed.UnmarshalText([]byte(level)); err != nil {
		parsed = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parsed}))
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ops := metrics.NewMongoOps(cfg.SlowQueryThreshold)
	db := database.NewManager(cfg, logger, ops)
	collector := metrics.NewCollector(ops, db.PoolUsage)
	cacheClient := cache.New(rootCtx, cfg.RedisAddr, cfg.RedisPassword, logger)
	defer cacheClient.Close()
	optimization := services.NewOptimizationService(db, cacheClient, logger)

	if err := applyInitialMode(rootCtx, optimization, cfg.InitialMode, logger); err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		db.Close(closeCtx)
	}()

	users := repository.NewUserRepository(db)
	productRepo := repository.NewProductRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	reviewRepo := repository.NewReviewRepository(db)
	resultRepo := repository.NewResultRepository(db)

	auth := services.NewAuthService(users, cfg.JWTSecret, cfg.JWTTTL, cfg.BcryptCost)
	adminCtx, cancelAdmin := context.WithTimeout(rootCtx, 15*time.Second)
	adminErr := auth.EnsureAdmin(adminCtx, cfg.AdminEmail, cfg.AdminPassword)
	cancelAdmin()
	if adminErr != nil {
		return adminErr
	}

	limiter := middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)
	go limiter.Run(rootCtx)
	go collector.Run(rootCtx)

	router := handlers.NewRouter(handlers.Dependencies{
		Config:       cfg,
		Logger:       logger,
		Auth:         auth,
		Products:     services.NewProductService(productRepo, cacheClient, optimization, cfg.ProductCacheTTL),
		Orders:       services.NewOrderService(orderRepo, productRepo, optimization),
		Reviews:      services.NewReviewService(reviewRepo, productRepo, cacheClient, optimization, cfg.SummaryCacheTTL),
		System:       services.NewSystemService(db, cacheClient, collector, ops, optimization),
		LoadTest:     services.NewLoadTestService(cfg.LoadTestBaseURL, users, productRepo, auth, resultRepo, optimization, db, logger),
		Optimization: optimization,
		Collector:    collector,
		Limiter:      limiter,
	})

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      cfg.RequestTimeout + writeTimeoutHeadroom,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("api listening", "address", server.Addr, "mode", string(cfg.InitialMode))
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			serverErrors <- serveErr
		}
	}()

	select {
	case serveErr := <-serverErrors:
		return serveErr
	case <-rootCtx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	logger.Info("shutdown complete")
	return nil
}

func applyInitialMode(ctx context.Context, optimization *services.OptimizationService, mode models.Mode, logger *slog.Logger) error {
	deadline := time.Now().Add(startupTimeout)
	for {
		attemptCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		_, err := optimization.Apply(attemptCtx, mode)
		cancel()
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return fmt.Errorf("initial mode %q could not be applied: %w", mode, err)
		}
		logger.Warn("waiting for mongodb", "error", err.Error())
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(startupRetryInterval):
		}
	}
}
