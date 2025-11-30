package usecase

import (
	"context"
	"stock_backend/internal/feature/candles/domain/entity"
)

const (
	DefaultInterval   = "1day"
	DefaultOutputSize = 200
	MaxOutputSize     = 5000
)

// CandleRepository abstracts the persistence layer for candlestick data.
// Following Go convention: interfaces are defined by the consumer (usecase), not the provider (adapters).
type CandleRepository interface {
	// Find searches the database for candlestick data.
	Find(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)

	// UpsertBatch performs upsert operations using (symbol, interval, time) as a unique key.
	UpsertBatch(ctx context.Context, candles []entity.Candle) error
}

// candlesUsecase defines the use case for candlestick data operations.
type candlesUsecase struct {
	candle CandleRepository
}

// NewCandlesUsecase creates a new candlesUsecase.
func NewCandlesUsecase(candle CandleRepository) *candlesUsecase {
	return &candlesUsecase{candle: candle}
}

// GetCandles retrieves candlestick data for a given symbol and time interval.
func (cu *candlesUsecase) GetCandles(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	if interval == "" {
		interval = DefaultInterval
	}
	if outputsize <= 0 || outputsize > MaxOutputSize {
		outputsize = DefaultOutputSize
	}

	cs, err := cu.candle.Find(ctx, symbol, interval, outputsize)
	if err != nil {
		return nil, err
	}

	return cs, nil
}
