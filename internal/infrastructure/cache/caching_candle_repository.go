package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"stock_backend/internal/domain/entity"
	"stock_backend/internal/domain/repository"
)

type CachingCandleRepository struct {
	inner     repository.CandleRepository
	rdb       *redis.Client
	ttl       time.Duration
	namespace string
}

// NewCachingCandleRepository は CandleRepository を Redis キャッシュでデコレートします。
// ttl=0 の場合は 5分にフォールバックします。namespace が空なら "candles" を使います。
func NewCachingCandleRepository(rdb *redis.Client, ttl time.Duration, inner repository.CandleRepository, namespace string) repository.CandleRepository {
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

func (c *CachingCandleRepository) UpsertBatch(ctx context.Context, candles []entity.Candle) error {
	// まず本体（MySQL）へ
	if err := c.inner.UpsertBatch(ctx, candles); err != nil {
		return err
	}
	// Redis 未設定なら終了
	if c.rdb == nil || len(candles) == 0 {
		return nil
	}

	// 影響範囲のキャッシュを無効化（symbol+interval ごとのキー）
	seen := map[string]struct{}{}
	for _, cd := range candles {
		prefix := c.cacheKeyPrefix(cd.Symbol, cd.Interval)
		if _, ok := seen[prefix]; ok {
			continue
		}
		seen[prefix] = struct{}{}
		_ = c.deleteByPattern(ctx, prefix+"*") // 失敗しても本処理は成功させる
	}
	return nil
}

func (c *CachingCandleRepository) Find(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	// Redis 未設定なら素通し
	if c.rdb == nil {
		return c.inner.Find(ctx, symbol, interval, outputsize)
	}

	key := c.cacheKey(symbol, interval, outputsize)

	// 1) キャッシュヒット確認
	if b, err := c.rdb.Get(ctx, key).Bytes(); err == nil && len(b) > 0 {
		var out []entity.Candle
		if err := json.Unmarshal(b, &out); err == nil {
			return out, nil
		}
		// 壊れていたら落とす
		_ = c.rdb.Del(ctx, key).Err()
	}

	// 2) DB へフォールバック
	out, err := c.inner.Find(ctx, symbol, interval, outputsize)
	if err != nil {
		return nil, err
	}

	// 3) キャッシュ保存（ベストエフォート）
	if b, err := json.Marshal(out); err == nil {
		_ = c.rdb.Set(ctx, key, b, c.ttl).Err()
	}

	return out, nil
}

// ---- 補助 ----

func (c *CachingCandleRepository) cacheKey(symbol, interval string, outputsize int) string {
	return fmt.Sprintf("%s:%s:%s:%d",
		c.namespace,
		safe(symbol),
		safe(interval),
		outputsize,
	)
}

func (c *CachingCandleRepository) cacheKeyPrefix(symbol, interval string) string {
	return fmt.Sprintf("%s:%s:%s:",
		c.namespace,
		safe(symbol),
		safe(interval),
	)
}

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

func safe(s string) string {
	// Redis キーに使いづらい記号の簡易エスケープ
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, ":", "_")
	return s
}
