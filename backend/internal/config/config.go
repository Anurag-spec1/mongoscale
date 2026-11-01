package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"mongoscale/internal/models"
)

type Seed struct {
	Users        int
	Products     int
	Orders       int
	Reviews      int
	BatchSize    int
	RandomSeed   int64
	Reset        bool
	ShardOrders  bool
	BcryptCost   int
	UserPassword string
}

type Config struct {
	HTTPPort           string
	LogLevel           string
	MongoURI           string
	MongoDatabase      string
	RedisAddr          string
	RedisPassword      string
	JWTSecret          string
	JWTTTL             time.Duration
	BcryptCost         int
	AdminEmail         string
	AdminPassword      string
	RequestTimeout     time.Duration
	MongoOpTimeout     time.Duration
	ShutdownTimeout    time.Duration
	SlowQueryThreshold time.Duration
	InitialMode        models.Mode
	RateLimitRPS       float64
	RateLimitBurst     int
	ProductCacheTTL    time.Duration
	SummaryCacheTTL    time.Duration
	CORSOrigin         string
	LoadTestBaseURL    string
	Seed               Seed
}

type loader struct {
	problems []string
}

func (l *loader) fail(key, value, expected string) {
	l.problems = append(l.problems, fmt.Sprintf("%s=%q is not a valid %s", key, value, expected))
}

func (l *loader) text(key, fallback string) string {
	value, found := os.LookupEnv(key)
	if !found || strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func (l *loader) integer(key string, fallback int) int {
	raw := l.text(key, "")
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		l.fail(key, raw, "integer")
		return fallback
	}
	return value
}

func (l *loader) integer64(key string, fallback int64) int64 {
	raw := l.text(key, "")
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		l.fail(key, raw, "integer")
		return fallback
	}
	return value
}

func (l *loader) float(key string, fallback float64) float64 {
	raw := l.text(key, "")
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		l.fail(key, raw, "number")
		return fallback
	}
	return value
}

func (l *loader) boolean(key string, fallback bool) bool {
	raw := l.text(key, "")
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		l.fail(key, raw, "boolean")
		return fallback
	}
	return value
}

func (l *loader) millis(key string, fallbackMillis int) time.Duration {
	return time.Duration(l.integer(key, fallbackMillis)) * time.Millisecond
}

func Load() (*Config, error) {
	l := &loader{}
	cfg := &Config{
		HTTPPort:           l.text("HTTP_PORT", "8080"),
		LogLevel:           l.text("LOG_LEVEL", "info"),
		MongoURI:           l.text("MONGO_URI", "mongodb://localhost:27017"),
		MongoDatabase:      l.text("MONGO_DATABASE", "mongoscale"),
		RedisAddr:          l.text("REDIS_ADDR", ""),
		RedisPassword:      l.text("REDIS_PASSWORD", ""),
		JWTSecret:          l.text("JWT_SECRET", ""),
		JWTTTL:             time.Duration(l.integer("JWT_TTL_MINUTES", 720)) * time.Minute,
		BcryptCost:         l.integer("BCRYPT_COST", 10),
		AdminEmail:         strings.ToLower(l.text("ADMIN_EMAIL", "admin@mongoscale.dev")),
		AdminPassword:      l.text("ADMIN_PASSWORD", ""),
		RequestTimeout:     l.millis("REQUEST_TIMEOUT_MS", 8000),
		MongoOpTimeout:     l.millis("MONGO_OP_TIMEOUT_MS", 5000),
		ShutdownTimeout:    l.millis("SHUTDOWN_TIMEOUT_MS", 15000),
		SlowQueryThreshold: l.millis("SLOW_QUERY_THRESHOLD_MS", 100),
		RateLimitRPS:       l.float("RATE_LIMIT_RPS", 30),
		RateLimitBurst:     l.integer("RATE_LIMIT_BURST", 60),
		ProductCacheTTL:    time.Duration(l.integer("PRODUCT_CACHE_TTL_SECONDS", 30)) * time.Second,
		SummaryCacheTTL:    time.Duration(l.integer("SUMMARY_CACHE_TTL_SECONDS", 60)) * time.Second,
		CORSOrigin:         l.text("CORS_ORIGIN", "*"),
		LoadTestBaseURL:    l.text("LOADTEST_BASE_URL", ""),
		Seed: Seed{
			Users:        l.integer("SEED_USERS", 5000),
			Products:     l.integer("SEED_PRODUCTS", 500),
			Orders:       l.integer("SEED_ORDERS", 50000),
			Reviews:      l.integer("SEED_REVIEWS", 100000),
			BatchSize:    l.integer("SEED_BATCH_SIZE", 5000),
			RandomSeed:   l.integer64("SEED_RANDOM_SEED", 42),
			Reset:        l.boolean("SEED_RESET", false),
			ShardOrders:  l.boolean("SEED_SHARD_ORDERS", false),
			BcryptCost:   l.integer("SEED_BCRYPT_COST", 8),
			UserPassword: l.text("SEED_USER_PASSWORD", "Password123!"),
		},
	}

	mode, err := models.ParseMode(l.text("INITIAL_MODE", string(models.ModeUnoptimized)))
	if err != nil {
		l.problems = append(l.problems, err.Error())
	}
	cfg.InitialMode = mode

	if cfg.LoadTestBaseURL == "" {
		cfg.LoadTestBaseURL = "http://127.0.0.1:" + cfg.HTTPPort
	}

	l.problems = append(l.problems, cfg.validate()...)
	if len(l.problems) > 0 {
		return nil, errors.New("invalid configuration: " + strings.Join(l.problems, "; "))
	}
	return cfg, nil
}

func (c *Config) validate() []string {
	var problems []string
	if len(c.JWTSecret) < 16 {
		problems = append(problems, "JWT_SECRET must be set and at least 16 characters")
	}
	if len(c.AdminPassword) < 8 {
		problems = append(problems, "ADMIN_PASSWORD must be set and at least 8 characters")
	}
	if c.BcryptCost < 4 || c.BcryptCost > 14 {
		problems = append(problems, "BCRYPT_COST must be between 4 and 14")
	}
	if c.Seed.BcryptCost < 4 || c.Seed.BcryptCost > 14 {
		problems = append(problems, "SEED_BCRYPT_COST must be between 4 and 14")
	}
	if c.Seed.BatchSize < 1 {
		problems = append(problems, "SEED_BATCH_SIZE must be positive")
	}
	if c.Seed.Users < 1 || c.Seed.Products < 1 || c.Seed.Orders < 0 || c.Seed.Reviews < 0 {
		problems = append(problems, "SEED_USERS and SEED_PRODUCTS must be positive; SEED_ORDERS and SEED_REVIEWS cannot be negative")
	}
	if c.RequestTimeout < time.Second || c.MongoOpTimeout < 100*time.Millisecond {
		problems = append(problems, "REQUEST_TIMEOUT_MS must be at least 1000 and MONGO_OP_TIMEOUT_MS at least 100")
	}
	return problems
}
