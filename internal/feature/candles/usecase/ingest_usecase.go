package usecase

import (
	"context"
	"log/slog"
	"stock_backend/internal/feature/candles/domain/entity"
	"stock_backend/internal/shared/ratelimiter"
)

const (
	ingestOutputSize = 200 // Number of data points to fetch per request
)

// ingestIntervals defines the list of time intervals for data ingestion.
var ingestIntervals = []string{"1day", "1week", "1month"}

// MarketRepository defines the repository interface for fetching stock market data.
// It abstracts external API implementations.
// Following Go convention: interfaces are defined by the consumer (usecase), not the provider (adapters).
type MarketRepository interface {
	GetTimeSeries(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
}

// IngestUsecase defines the use case for fetching data from external APIs
// and persisting it to the database.
type IngestUsecase struct {
	market      MarketRepository
	candle      CandleRepository
	rateLimiter ratelimiter.RateLimiterInterface
}

// NewIngestUsecase creates a new IngestUsecase.
func NewIngestUsecase(market MarketRepository, candle CandleRepository, rateLimiter ratelimiter.RateLimiterInterface) *IngestUsecase {
	return &IngestUsecase{market: market, candle: candle, rateLimiter: rateLimiter}
}

// ingestOne fetches time series data for a specified symbol and interval from
// the external repository and batch inserts (or updates) it into the database.
func (iu *IngestUsecase) ingestOne(ctx context.Context, symbol, interval string, outputsize int) error {
	cs, err := iu.market.GetTimeSeries(ctx, symbol, interval, outputsize)
	if err != nil {
		return err
	}

	// Set symbol and interval for the fetched data
	for i := range cs {
		cs[i].Symbol = symbol
		cs[i].Interval = interval
	}
	return iu.candle.UpsertBatch(ctx, cs)
}

// IngestAll fetches time series data for all specified symbols across multiple
// time intervals (daily, weekly, monthly) and persists them to the database.
// It respects API rate limits by waiting between requests as needed.
func (iu *IngestUsecase) IngestAll(ctx context.Context, symbols []string) error {
	for _, s := range symbols {
		for _, interval := range ingestIntervals {
			iu.rateLimiter.WaitIfNeeded()
			if err := iu.ingestOne(ctx, s, interval, ingestOutputSize); err != nil {
				// Continue processing even if one symbol fails, logging the error
				slog.Error("failed to ingest data", "symbol", s, "interval", interval, "error", err)
				continue // Move to next interval or symbol
			}
		}
	}
	return nil
}
