package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"stock_backend/internal/feature/candles/domain/entity"
)

// maxCacheOutputSize はキャッシュに保存する最大ローソク足件数です。
// usecaseのMaxOutputSizeと合わせています（依存ルール上usecaseをimport不可のためここで定義）。
const maxCacheOutputSize = 5000

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

// UpsertBatch はローソク足データを挿入または更新し、キャッシュを最新データで更新します。
func (c *CachingCandleRepository) UpsertBatch(ctx context.Context, candles []entity.Candle) error {
	// まず基盤リポジトリにUpsert
	if err := c.inner.UpsertBatch(ctx, candles); err != nil {
		return err
	}
	// Redisが未設定またはデータがない場合は早期リターン
	if c.rdb == nil || len(candles) == 0 {
		return nil
	}

	// 影響を受ける symbol+interval を収集
	type symbolInterval struct {
		symbol   string
		interval string
	}
	seen := map[symbolInterval]struct{}{}
	for _, cd := range candles {
		seen[symbolInterval{cd.Symbol, cd.Interval}] = struct{}{}
	}

	// 各 symbol+interval のキャッシュを削除し、最新データで再生成（ウォームアップ）
	for si := range seen {
		key := c.cacheKey(si.symbol, si.interval)
		_ = c.rdb.Del(ctx, key).Err() // ベストエフォート

		data, err := c.inner.Find(ctx, si.symbol, si.interval, maxCacheOutputSize)
		if err != nil {
			continue // ベストエフォート: エラー時はウォームアップをスキップ
		}
		if b, err := json.Marshal(data); err == nil {
			_ = c.rdb.Set(ctx, key, b, c.ttl).Err() // ベストエフォート
		}
	}
	return nil
}

// Find はローソク足データを取得します。まずキャッシュを確認し、なければデータベースにフォールバックします。
// キャッシュには全データ（最大maxCacheOutputSize件）を保存し、outputsize件にスライスして返します。
func (c *CachingCandleRepository) Find(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	// Redisが未設定の場合はキャッシュをバイパス
	if c.rdb == nil {
		return c.inner.Find(ctx, symbol, interval, outputsize)
	}

	key := c.cacheKey(symbol, interval)

	// 1) キャッシュを確認
	if b, err := c.rdb.Get(ctx, key).Bytes(); err == nil && len(b) > 0 {
		var all []entity.Candle
		if err := json.Unmarshal(b, &all); err == nil {
			return sliceCandles(all, outputsize), nil
		}
		// 破損したキャッシュエントリを削除
		_ = c.rdb.Del(ctx, key).Err()
	}

	// 2) データベースにフォールバック（全データ取得してキャッシュに保存）
	all, err := c.inner.Find(ctx, symbol, interval, maxCacheOutputSize)
	if err != nil {
		return nil, err
	}

	// 3) キャッシュに保存（ベストエフォート）
	if b, err := json.Marshal(all); err == nil {
		_ = c.rdb.Set(ctx, key, b, c.ttl).Err()
	}

	return sliceCandles(all, outputsize), nil
}

// sliceCandles は全ローソク足データから先頭 outputsize 件を返します。
func sliceCandles(all []entity.Candle, outputsize int) []entity.Candle {
	if outputsize <= 0 || outputsize >= len(all) {
		return all
	}
	return all[:outputsize]
}

// cacheKey はキャッシュキーを生成します。
func (c *CachingCandleRepository) cacheKey(symbol, interval string) string {
	return fmt.Sprintf("%s:%s:%s",
		c.namespace,
		safeCacheKey(symbol),
		safeCacheKey(interval),
	)
}

// safeCacheKey はRedisキーで問題となる文字をエスケープします。
func safeCacheKey(s string) string {
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, ":", "_")
	return s
}
