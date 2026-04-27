package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	redisv9 "github.com/redis/go-redis/v9"

	"stock_backend/internal/app/di"
	candlesadapters "stock_backend/internal/feature/candles/adapters"
	candlesusecase "stock_backend/internal/feature/candles/usecase"
	symbollistadapters "stock_backend/internal/feature/symbollist/adapters"
	"stock_backend/internal/platform/db"
	infraredis "stock_backend/internal/platform/redis"
	"stock_backend/internal/shared/ratelimiter"
)

const (
	rateLimitPerMinute    = 7   // TwelveData APIのレートリミット（無料枠上限8/分、固定ウィンドウずれ対策で1つ余裕を持たせる）
	defaultMaxFailureRate = 0.2 // INGEST_MAX_FAILURE_RATE のデフォルト値
)

// shouldFailExit は ingest サマリと失敗率しきい値から非ゼロ終了すべきかを判定する。
// しきい値ちょうど（FailureRate == threshold）は許容し、超過時のみ true を返す。
func shouldFailExit(result candlesusecase.IngestResult, threshold float64) bool {
	return result.FailureRate() > threshold
}

func main() {
	db := db.OpenDB()
	marketRepo := di.NewMarket()
	candleRepo := candlesadapters.NewCandleRepository(db)
	symbolRepo := symbollistadapters.NewSymbolRepository(db)
	rateLimiter := ratelimiter.NewRateLimiter(rateLimitPerMinute, time.Minute)

	// Redis接続（ベストエフォート: 接続失敗時はキャッシュウォームアップなしで続行）
	var rdb *redisv9.Client
	if tmp, err := infraredis.NewRedisClient(); err != nil {
		slog.Warn("Redis unavailable, cache warm-up disabled", "error", err)
	} else {
		rdb = tmp
		defer func() {
			if err := rdb.Close(); err != nil {
				slog.Error("Failed to close Redis client", "error", err)
			}
		}()
	}

	// TTLはingest連続失敗時のセーフティネット、通常は UpsertBatch で日次上書き
	cachedCandleRepo := candlesadapters.NewCachingCandleRepository(rdb, candlesadapters.DefaultCacheTTL, candleRepo, "candles")

	uc := candlesusecase.NewIngestUsecase(marketRepo, cachedCandleRepo, symbolRepo, rateLimiter)

	timeoutHours := 3
	if v := os.Getenv("INGEST_TIMEOUT_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutHours = n
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutHours)*time.Hour)
	defer cancel()

	maxFailureRate := defaultMaxFailureRate
	if v := os.Getenv("INGEST_MAX_FAILURE_RATE"); v != "" {
		if r, err := strconv.ParseFloat(v, 64); err == nil && r >= 0 && r <= 1 {
			maxFailureRate = r
		} else {
			slog.Warn("invalid INGEST_MAX_FAILURE_RATE, using default", "value", v, "default", defaultMaxFailureRate)
		}
	}

	start := time.Now()
	result, err := uc.IngestAll(ctx)
	duration := time.Since(start)

	slog.Info("ingest summary",
		"total", result.Total,
		"succeeded", result.Succeeded,
		"failed", result.Failed,
		"failure_rate", result.FailureRate(),
		"duration", duration.String(),
	)

	if err != nil {
		slog.Error("ingest aborted by fatal error", "error", err)
		os.Exit(1)
	}
	if shouldFailExit(result, maxFailureRate) {
		slog.Error("ingest failure rate exceeded threshold",
			"failure_rate", result.FailureRate(),
			"threshold", maxFailureRate,
		)
		os.Exit(1)
	}
	slog.Info("ingest ok")
}
