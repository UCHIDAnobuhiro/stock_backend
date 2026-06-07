package batch

import (
	"context"
	"log/slog"
	"time"

	redisv9 "github.com/redis/go-redis/v9"

	"github.com/UCHIDAnobuhiro/stock-backend/internal/app/di"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/candles"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/symbollist"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/infra/db"
	infraredis "github.com/UCHIDAnobuhiro/stock-backend/internal/infra/redis"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/shared/clientratelimit"
)

// runCandleIngest は TwelveData から株価データを取り込み、終了コード（0 or 1）を返す。
func runCandleIngest() int {
	sqlDB, err := db.OpenSQL()
	if err != nil {
		slog.Error("DB open failed", "error", err)
		return 1
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			slog.Warn("failed to close sqlDB", "error", err)
		}
	}()
	marketRepo := di.NewMarket()
	candleRepo := candles.NewRepository(sqlDB)
	symbolRepo := symbollist.NewRepository(sqlDB)
	ingestSymbolRepo := di.NewIngestSymbolAdapter(symbolRepo)
	rateLimiter := clientratelimit.NewRateLimiter(rateLimitPerMinute, time.Minute)

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
	cachedCandleRepo := candles.NewCachingRepository(rdb, candles.DefaultCacheTTL, candleRepo, "candles")

	uc := candles.NewIngestUsecase(marketRepo, cachedCandleRepo, ingestSymbolRepo, rateLimiter)

	timeoutHours := parseTimeoutHours("INGEST_TIMEOUT_HOURS", 3)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutHours)*time.Hour)
	defer cancel()

	maxFailureRate := parseMaxFailureRate("INGEST_MAX_FAILURE_RATE", defaultMaxFailureRate)

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
		return 1
	}
	if shouldFailExit(result, maxFailureRate) {
		slog.Error("ingest failure rate exceeded threshold",
			"failure_rate", result.FailureRate(),
			"threshold", maxFailureRate,
		)
		return 1
	}
	slog.Info("ingest ok")
	return 0
}
