package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"strconv"
	"time"

	"stock_backend/internal/app/di"
	candlesadapters "stock_backend/internal/feature/candles/adapters"
	candlesusecase "stock_backend/internal/feature/candles/usecase"
	symbollistadapters "stock_backend/internal/feature/symbollist/adapters"
	"stock_backend/internal/platform/cache"
	"stock_backend/internal/platform/db"
	infraredis "stock_backend/internal/platform/redis"
	"stock_backend/internal/shared/ratelimiter"
)

const (
	rateLimitPerMinute = 7 // TwelveData APIのレートリミット（無料枠上限8/分、固定ウィンドウずれ対策で1つ余裕を持たせる）
)

func main() {
	db := db.OpenDB()
	marketRepo := di.NewMarket()
	candleRepo := candlesadapters.NewCandleRepository(db)
	symbolRepo := symbollistadapters.NewSymbolRepository(db)
	rateLimiter := ratelimiter.NewRateLimiter(rateLimitPerMinute, time.Minute)

	// Redis接続（ベストエフォート: 接続失敗時はキャッシュウォームアップなしで続行）
	var rdb interface{ Close() error }
	ttl := cache.TimeUntilNext8AM()
	cachedCandleRepo := candlesadapters.NewCachingCandleRepository(nil, ttl, candleRepo, "candles")
	if tmp, err := infraredis.NewRedisClient(); err != nil {
		slog.Warn("Redis unavailable, cache warm-up disabled", "error", err)
	} else {
		rdb = tmp
		defer func() {
			if err := rdb.Close(); err != nil {
				slog.Error("Failed to close Redis client", "error", err)
			}
		}()
		cachedCandleRepo = candlesadapters.NewCachingCandleRepository(tmp, ttl, candleRepo, "candles")
	}

	uc := candlesusecase.NewIngestUsecase(marketRepo, cachedCandleRepo, symbolRepo, rateLimiter)

	timeoutHours := 3
	if v := os.Getenv("INGEST_TIMEOUT_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutHours = n
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutHours)*time.Hour)
	defer cancel()

	if err := uc.IngestAll(ctx); err != nil {
		log.Fatal(err)
	}
	log.Println("ingest ok")
}
