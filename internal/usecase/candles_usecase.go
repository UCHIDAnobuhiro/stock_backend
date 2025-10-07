package usecase

import (
	"context"
	"stock_backend/internal/domain/entity"
	"stock_backend/internal/domain/repository"
)

const (
	DefaultInterval   = "1day"
	DefaultOutputSize = 200
	MaxOutputSize     = 5000
)

// CandlesUsecase はロウソク足データに関するユースケースを定義します。
type CandlesUsecase struct {
	candle repository.CandleRepository
}

// NewCandlesUsecase は新しい CandlesUsecase を作成します。
func NewCandlesUsecase(candle repository.CandleRepository) *CandlesUsecase {
	return &CandlesUsecase{candle: candle}
}

// GetCandles は銘柄コードと時間足(interval)を指定してロウソク足データを取得します。
func (cu *CandlesUsecase) GetCandles(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
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
