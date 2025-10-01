package main

import (
	"context"
	"log"
	"stock_backend/internal/infrastructure"
	"stock_backend/internal/infrastructure/db"
	"stock_backend/internal/infrastructure/mysql"
	"stock_backend/internal/usecase"
	"time"
)

func main() {

	db := db.OpenDB()
	marketRepo := infrastructure.NewMarket()
	candleRepo := mysql.NewCandleRepository(db)
	symbolRepo := mysql.NewSymbolRepository(db)
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
