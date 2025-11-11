package main

import (
	"context"
	"log"
	candlesadapters "stock_backend/internal/feature/candles/adapters"
	symbollistadapters "stock_backend/internal/feature/symbollist/adapters"
	"stock_backend/internal/infrastructure"
	"stock_backend/internal/infrastructure/db"
	"stock_backend/internal/usecase"
	"time"
)

func main() {

	db := db.OpenDB()
	marketRepo := infrastructure.NewMarket()
	candleRepo := candlesadapters.NewCandleRepository(db)
	symbolRepo := symbollistadapters.NewSymbolRepository(db)
	uc := usecase.NewIngestUsecase(marketRepo, candleRepo)

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
