// Package cache はリポジトリインターフェースのキャッシュ実装を提供します。
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"stock_backend/internal/feature/candles/domain/entity"
)

// candleRepository はCachingCandleRepositoryが内部で必要とする読み書きインターフェースです。
type candleRepository interface {
	Find(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
	UpsertBatch(ctx context.Context, candles []entity.Candle) error
}

// CachingCandleRepository はCandleRepositoryにRedisキャッシュをデコレータパターンで追加します。
// 基盤となるリポジトリを変更せずに、透過的にキャッシュを追加します。
type CachingCandleRepository struct {
	inner     candleRepository
	rdb       *redis.Client
	ttl       time.Duration
	namespace string
}

// NewCachingCandleRepository はCandleRepositoryにRedisキャッシュを追加するデコレータを生成します。
// ttlが0の場合はデフォルト5分、namespaceが空の場合は"candles"を使用します。
func NewCachingCandleRepository(rdb *redis.Client, ttl time.Duration, inner candleRepository, namespace string) *CachingCandleRepository {
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

// UpsertBatch はローソク足データを挿入または更新し、関連するキャッシュエントリを無効化します。
func (c *CachingCandleRepository) UpsertBatch(ctx context.Context, candles []entity.Candle) error {
	// まず基盤リポジトリ（MySQL）にUpsert
	if err := c.inner.UpsertBatch(ctx, candles); err != nil {
		return err
	}
	// Redisが未設定またはデータがない場合は早期リターン
	if c.rdb == nil || len(candles) == 0 {
		return nil
	}

	// 影響を受けるキャッシュエントリを無効化（symbol+intervalごとのキー）
	seen := map[string]struct{}{}
	for _, cd := range candles {
		prefix := c.cacheKeyPrefix(cd.Symbol, cd.Interval)
		if _, ok := seen[prefix]; ok {
			continue
		}
		seen[prefix] = struct{}{}
		_ = c.deleteByPattern(ctx, prefix+"*") // ベストエフォート: キャッシュ削除失敗時もエラーにしない
	}
	return nil
}

// Find はローソク足データを取得します。まずキャッシュを確認し、なければデータベースにフォールバックします。
func (c *CachingCandleRepository) Find(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	// Redisが未設定の場合はキャッシュをバイパス
	if c.rdb == nil {
		return c.inner.Find(ctx, symbol, interval, outputsize)
	}

	key := c.cacheKey(symbol, interval, outputsize)

	// 1) キャッシュを確認
	if b, err := c.rdb.Get(ctx, key).Bytes(); err == nil && len(b) > 0 {
		var out []entity.Candle
		if err := json.Unmarshal(b, &out); err == nil {
			return out, nil
		}
		// 破損したキャッシュエントリを削除
		_ = c.rdb.Del(ctx, key).Err()
	}

	// 2) データベースにフォールバック
	out, err := c.inner.Find(ctx, symbol, interval, outputsize)
	if err != nil {
		return nil, err
	}

	// 3) キャッシュに保存（ベストエフォート）
	if b, err := json.Marshal(out); err == nil {
		_ = c.rdb.Set(ctx, key, b, c.ttl).Err()
	}

	return out, nil
}

// cacheKey は特定のクエリ用のキャッシュキーを生成します。
func (c *CachingCandleRepository) cacheKey(symbol, interval string, outputsize int) string {
	return fmt.Sprintf("%s:%s:%s:%d",
		c.namespace,
		safe(symbol),
		safe(interval),
		outputsize,
	)
}

// cacheKeyPrefix は関連するキャッシュエントリの無効化用プレフィックスを生成します。
func (c *CachingCandleRepository) cacheKeyPrefix(symbol, interval string) string {
	return fmt.Sprintf("%s:%s:%s:",
		c.namespace,
		safe(symbol),
		safe(interval),
	)
}

// deleteByPattern はSCANを使用して指定パターンに一致するすべてのキャッシュキーを削除します。
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

// safe はRedisキーで問題となる文字をエスケープします。
func safe(s string) string {
	// Redisキーで問題となる文字の簡易エスケープ
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, ":", "_")
	return s
}
