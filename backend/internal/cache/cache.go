package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

const operationTimeout = 300 * time.Millisecond

type Stats struct {
	Enabled bool  `json:"enabled"`
	Hits    int64 `json:"hits"`
	Misses  int64 `json:"misses"`
}

type Cache struct {
	client *redis.Client
	logger *slog.Logger
	hits   atomic.Int64
	misses atomic.Int64
}

func New(ctx context.Context, address, password string, logger *slog.Logger) *Cache {
	if address == "" {
		logger.Info("redis cache disabled: REDIS_ADDR is empty")
		return &Cache{logger: logger}
	}
	client := redis.NewClient(&redis.Options{
		Addr:         address,
		Password:     password,
		PoolSize:     300,
		MinIdleConns: 10,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  operationTimeout,
		WriteTimeout: operationTimeout,
	})
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		logger.Warn("redis unreachable, caching disabled", "error", err.Error())
		_ = client.Close()
		return &Cache{logger: logger}
	}
	logger.Info("redis cache connected", "address", address)
	return &Cache{client: client, logger: logger}
}

func (c *Cache) Enabled() bool {
	return c.client != nil
}

func (c *Cache) Ping(ctx context.Context) error {
	if c.client == nil {
		return errors.New("redis disabled")
	}
	pingCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return c.client.Ping(pingCtx).Err()
}

func (c *Cache) GetJSON(ctx context.Context, key string, destination any) bool {
	if c.client == nil {
		return false
	}
	opCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	payload, err := c.client.Get(opCtx, key).Bytes()
	if err != nil {
		c.misses.Add(1)
		if !errors.Is(err, redis.Nil) {
			c.logger.Debug("cache read failed", "key", key, "error", err.Error())
		}
		return false
	}
	if err := json.Unmarshal(payload, destination); err != nil {
		c.misses.Add(1)
		return false
	}
	c.hits.Add(1)
	return true
}

func (c *Cache) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) {
	if c.client == nil {
		return
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return
	}
	opCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	if err := c.client.Set(opCtx, key, payload, ttl).Err(); err != nil {
		c.logger.Debug("cache write failed", "key", key, "error", err.Error())
	}
}

func (c *Cache) Delete(ctx context.Context, keys ...string) {
	if c.client == nil || len(keys) == 0 {
		return
	}
	opCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	if err := c.client.Del(opCtx, keys...).Err(); err != nil {
		c.logger.Debug("cache delete failed", "error", err.Error())
	}
}

func (c *Cache) Version(ctx context.Context, namespace string) string {
	if c.client == nil {
		return "0"
	}
	opCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	value, err := c.client.Get(opCtx, "version:"+namespace).Int64()
	if err != nil {
		return "0"
	}
	return strconv.FormatInt(value, 10)
}

func (c *Cache) BumpVersion(ctx context.Context, namespace string) {
	if c.client == nil {
		return
	}
	opCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	if err := c.client.Incr(opCtx, "version:"+namespace).Err(); err != nil {
		c.logger.Debug("cache version bump failed", "error", err.Error())
	}
}

func (c *Cache) Flush(ctx context.Context) {
	if c.client == nil {
		return
	}
	opCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := c.client.FlushDB(opCtx).Err(); err != nil {
		c.logger.Warn("cache flush failed", "error", err.Error())
	}
}

func (c *Cache) Stats() Stats {
	return Stats{Enabled: c.client != nil, Hits: c.hits.Load(), Misses: c.misses.Load()}
}

func (c *Cache) Close() {
	if c.client != nil {
		_ = c.client.Close()
	}
}
