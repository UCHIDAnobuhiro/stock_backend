package main

import (
	"context"
	"log"
	"stock_backend/internal/infrastructure"
	"stock_backend/internal/infrastructure/mysql"
	"stock_backend/internal/usecase"
	"time"
)

func main() {
	db := infrastructure.OpenDB()
	market := infrastructure.NewMarket()
	candle := mysql.NewCandleRepository(db)
	uc := usecase.NewIngestUsecase(market, candle)

	symbols := []string{"AAPL", "MSFT", "GOOGL"} // 管理テーブル or 設定から読み込む
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := uc.IngestAll(ctx, symbols); err != nil {
		log.Fatal(err)
	}
	log.Println("ingest ok")
}
