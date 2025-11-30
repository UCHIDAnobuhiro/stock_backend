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

// CandleRepository はロウソク足の永続化を抽象化します。
// Following Go convention: interfaces are defined by the consumer (usecase), not the provider (adapters).
type CandleRepository interface {
	// Find はデータベースを検索します。
	Find(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)

	// UpsertBatch は (symbol, interval, time) をユニークキーとしてUpsertします。
	UpsertBatch(ctx context.Context, candles []entity.Candle) error
}

// candlesUsecase はロウソク足データに関するユースケースを定義します。
type candlesUsecase struct {
	candle CandleRepository
}

// NewCandlesUsecase は新しい candlesUsecase を作成します。
func NewCandlesUsecase(candle CandleRepository) *candlesUsecase {
	return &candlesUsecase{candle: candle}
}

// GetCandles は銘柄コードと時間足(interval)を指定してロウソク足データを取得します。
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
