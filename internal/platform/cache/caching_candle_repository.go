// Package cache provides caching implementations for repository interfaces.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"stock_backend/internal/feature/candles/domain/entity"
	"stock_backend/internal/feature/candles/usecase"
)

// CachingCandleRepository decorates a CandleRepository with Redis caching.
// It implements the decorator pattern, transparently adding caching without
// modifying the underlying repository.
type CachingCandleRepository struct {
	inner     usecase.CandleRepository
	rdb       *redis.Client
	ttl       time.Duration
	namespace string
}

// NewCachingCandleRepository decorates a CandleRepository with Redis caching.
// If ttl is 0, it defaults to 5 minutes. If namespace is empty, it uses "candles".
func NewCachingCandleRepository(rdb *redis.Client, ttl time.Duration, inner usecase.CandleRepository, namespace string) *CachingCandleRepository {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	if namespace == "" {
		namespace = "candles"
	}
	return &CachingCandleRepository{
		inner:     inner,
		rdb:       rdb,
		ttl:       ttl,
		namespace: namespace,
	}
}

// UpsertBatch inserts or updates candles and invalidates related cache entries.
func (c *CachingCandleRepository) UpsertBatch(ctx context.Context, candles []entity.Candle) error {
	// First upsert to the underlying repository (MySQL)
	if err := c.inner.UpsertBatch(ctx, candles); err != nil {
		return err
	}
	// Exit early if Redis is not configured or there are no candles
	if c.rdb == nil || len(candles) == 0 {
		return nil
	}

	// Invalidate affected cache entries (keys per symbol+interval)
	seen := map[string]struct{}{}
	for _, cd := range candles {
		prefix := c.cacheKeyPrefix(cd.Symbol, cd.Interval)
		if _, ok := seen[prefix]; ok {
			continue
		}
		seen[prefix] = struct{}{}
		_ = c.deleteByPattern(ctx, prefix+"*") // Best effort: don't fail if cache deletion fails
	}
	return nil
}

// Find retrieves candles, checking cache first then falling back to the database.
func (c *CachingCandleRepository) Find(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	// Bypass cache if Redis is not configured
	if c.rdb == nil {
		return c.inner.Find(ctx, symbol, interval, outputsize)
	}

	key := c.cacheKey(symbol, interval, outputsize)

	// 1) Check cache
	if b, err := c.rdb.Get(ctx, key).Bytes(); err == nil && len(b) > 0 {
		var out []entity.Candle
		if err := json.Unmarshal(b, &out); err == nil {
			return out, nil
		}
		// Delete corrupted cache entry
		_ = c.rdb.Del(ctx, key).Err()
	}

	// 2) Fallback to database
	out, err := c.inner.Find(ctx, symbol, interval, outputsize)
	if err != nil {
		return nil, err
	}

	// 3) Store in cache (best effort)
	if b, err := json.Marshal(out); err == nil {
		_ = c.rdb.Set(ctx, key, b, c.ttl).Err()
	}

	return out, nil
}

// cacheKey generates a cache key for a specific query.
func (c *CachingCandleRepository) cacheKey(symbol, interval string, outputsize int) string {
	return fmt.Sprintf("%s:%s:%s:%d",
		c.namespace,
		safe(symbol),
		safe(interval),
		outputsize,
	)
}

// cacheKeyPrefix generates a prefix for invalidating related cache entries.
func (c *CachingCandleRepository) cacheKeyPrefix(symbol, interval string) string {
	return fmt.Sprintf("%s:%s:%s:",
		c.namespace,
		safe(symbol),
		safe(interval),
	)
}

// deleteByPattern deletes all cache keys matching a given pattern using SCAN.
func (c *CachingCandleRepository) deleteByPattern(ctx context.Context, pattern string) error {
	var cursor uint64
	for {
		keys, cur, err := c.rdb.Scan(ctx, cursor, pattern, 200).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = cur
		if cursor == 0 {
			break
		}
	}
	return nil
}

// safe escapes characters that are problematic for Redis keys.
func safe(s string) string {
	// Simple escaping of characters that are problematic for Redis keys
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, ":", "_")
	return s
}
