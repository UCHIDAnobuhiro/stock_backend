package usecase

import (
	"context"
	"stock_backend/internal/domain/entity"
	"stock_backend/internal/domain/repository"
)

type CandlesUsecase struct {
	market repository.MarketRepository
	candle repository.CandleRepository
}

func NewCandlesUsecase(market repository.MarketRepository, candle repository.CandleRepository) *CandlesUsecase {
	return &CandlesUsecase{market: market, candle: candle}
}

// GetCandles は銘柄コードと時間足(interval)を指定してロウソク足データを取得します。
func (cu *CandlesUsecase) GetCandles(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	if interval == "" {
		interval = "1day"
	}
	if outputsize <= 0 || outputsize > 5000 {
		outputsize = 200
	}

	cs, err := cu.candle.Find(ctx, symbol, interval, outputsize)
	if err != nil {
		return nil, err
	}

	return cs, nil
}
