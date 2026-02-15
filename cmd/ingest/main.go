package main

import (
	"context"
	"log"
	"time"

	"stock_backend/internal/app/di"
	candlesadapters "stock_backend/internal/feature/candles/adapters"
	candlesusecase "stock_backend/internal/feature/candles/usecase"
	symbollistadapters "stock_backend/internal/feature/symbollist/adapters"
	"stock_backend/internal/platform/db"
	"stock_backend/internal/shared/ratelimiter"
)

const (
	rateLimitPerMinute = 8 // TwelveData APIのレートリミット（無料枠: 8リクエスト/分）
)

func main() {
	db := db.OpenDB()
	marketRepo := di.NewMarket()
	candleRepo := candlesadapters.NewCandleRepository(db)
	symbolRepo := symbollistadapters.NewSymbolRepository(db)
	rateLimiter := ratelimiter.NewRateLimiter(rateLimitPerMinute, time.Minute)

	uc := candlesusecase.NewIngestUsecase(marketRepo, candleRepo, symbolRepo, rateLimiter)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := uc.IngestAll(ctx); err != nil {
		log.Fatal(err)
	}
	log.Println("ingest ok")
}
