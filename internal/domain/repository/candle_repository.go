package repository

import (
	"context"
	"stock_backend/internal/domain/entity"
)

// CandleRepository　はロウソク足の永続化を抽象化します
type CandleRepository interface {
	// データベースを検索
	Find(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)

	// (symbol, interval, time) をユニークキーとしてUpsert
	UpsertBatch(ctx context.Context, candles []entity.Candle) error
}
