package usecase

import (
	"context"
	"stock_backend/internal/domain/repository"
	"time"
)

type IngestUsecase struct {
	market repository.MarketRepository
	candle repository.CandleRepository
}

func NewIngestUsecase(market repository.MarketRepository, candle repository.CandleRepository) *IngestUsecase {
	return &IngestUsecase{market: market, candle: candle}
}

func (iu *IngestUsecase) ingestOne(ctx context.Context, symbol, interval string, outputsize int) error {
	cs, err := iu.market.GetTimeSeries(ctx, symbol, interval, outputsize)
	if err != nil {
		return err
	}

	for i := range cs {
		cs[i].Symbol = symbol
		cs[i].Interval = interval
	}
	return iu.candle.UpsertBatch(ctx, cs)
}

func (iu *IngestUsecase) IngestAll(ctx context.Context, symbols []string) error {
	rl := NewRateLimiter(8, time.Minute) // 1分に8回まで
	for _, s := range symbols {
		for _, interval := range []string{"1day", "1week", "1month"} {
			rl.WaitIfNeeded()
			if err := iu.ingestOne(ctx, s, interval, 200); err != nil {
				return err
			}
		}
	}
	return nil
}
