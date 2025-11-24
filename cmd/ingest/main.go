package main

import (
	"context"
	"log"
	"stock_backend/internal/app/di"
	candlesadapters "stock_backend/internal/feature/candles/adapters"
	candlesusecase "stock_backend/internal/feature/candles/usecase"
	symbollistadapters "stock_backend/internal/feature/symbollist/adapters"
	"stock_backend/internal/platform/db"
	"stock_backend/internal/shared/ratelimiter"
	"time"
)

const (
	rateLimitPerMinute = 8 // TwelveData APIの1分あたりのリクエスト上限（フリープラン）
)

func main() {

	db := db.OpenDB()
	marketRepo := di.NewMarket()
	candleRepo := candlesadapters.NewCandleRepository(db)
	symbolRepo := symbollistadapters.NewSymbolRepository(db)
	rateLimiter := ratelimiter.NewRateLimiter(rateLimitPerMinute, time.Minute)

	uc := candlesusecase.NewIngestUsecase(marketRepo, candleRepo, rateLimiter)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	symbols, err := symbolRepo.ListActiveCodes(ctx)
	if err != nil {
		log.Fatal("failed to load symbols:", err)
	}

	if err := uc.IngestAll(ctx, symbols); err != nil {
		log.Fatal(err)
	}
	log.Println("ingest ok")
}
