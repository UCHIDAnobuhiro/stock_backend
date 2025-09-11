package usecase

import (
	"context"
	"todo_backend/internal/domain/entity"
	"todo_backend/internal/domain/repository"
)

type CandlesUsecase struct {
	market repository.MarketRepository
}

func NewCandlesUsecase(market repository.MarketRepository) *CandlesUsecase {
	return &CandlesUsecase{market: market}
}

// GetCandles は銘柄コードと時間足(interval)を指定してロウソク足データを取得します。
func (cu *CandlesUsecase) GetCandles(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	if interval == "" {
		interval = "1day"
	}
	if outputsize <= 0 || outputsize > 5000 {
		outputsize = 200
	}

	return cu.market.GetTimeSeries(ctx, symbol, interval, outputsize)
}
