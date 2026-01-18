package twelvedata

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"stock_backend/internal/feature/candles/domain/entity"
	"stock_backend/internal/feature/candles/usecase"
	"stock_backend/internal/platform/externalapi/twelvedata/dto"
	"strconv"
	"time"
)

// TwelveDataMarket is a MarketRepository implementation that fetches stock data
// from the Twelve Data external API.
type TwelveDataMarket struct {
	cfg    Config
	client *http.Client
}

// Compile-time check to ensure TwelveDataMarket implements MarketRepository.
var _ usecase.MarketRepository = (*TwelveDataMarket)(nil)

// NewTwelveDataMarket creates a new TwelveDataMarket with the given config and HTTP client.
func NewTwelveDataMarket(cfg Config, client *http.Client) *TwelveDataMarket {
	return &TwelveDataMarket{cfg: cfg, client: client}
}

// GetTimeSeries retrieves time series stock data from the Twelve Data API
// and returns it as a slice of domain.Candle.
func (t *TwelveDataMarket) GetTimeSeries(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	q := url.Values{}
	// Add query parameters
	q.Set("symbol", symbol)
	q.Set("interval", interval)
	q.Set("outputsize", strconv.Itoa(outputsize))
	q.Set("apikey", t.cfg.TwelveDataAPIKey)

	// Generate URL
	u := fmt.Sprintf("%s/time_series?%s", t.cfg.BaseURL, q.Encode())

	// Create request object
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	// Execute request
	res, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			slog.Warn("failed to close response body", "error", err)
		}
	}()

	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("twelvedata http %d", res.StatusCode)
	}

	// Decode JSON response into DTO
	var body dto.TimeSeriesResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.Status == "error" {
		return nil, fmt.Errorf("twelvedata: %s", body.Message)
	}

	candles := make([]entity.Candle, 0, len(body.Values))
	for _, v := range body.Values {

		// Parse timestamp
		tm, err := time.Parse("2006-01-02 15:04:05", v.Datetime)
		if err != nil {
			tm, err = time.Parse("2006-01-02", v.Datetime)
			if err != nil {
				return nil, fmt.Errorf("parse time %q: %w", v.Datetime, err)
			}
		}
		// Parse open price
		o, err := strconv.ParseFloat(v.Open, 64)
		if err != nil {
			return nil, fmt.Errorf("parse open %q: %w", v.Open, err)
		}
		// Parse high price
		h, err := strconv.ParseFloat(v.High, 64)
		if err != nil {
			return nil, fmt.Errorf("parse high %q: %w", v.High, err)
		}
		// Parse low price
		l, err := strconv.ParseFloat(v.Low, 64)
		if err != nil {
			return nil, fmt.Errorf("parse low %q: %w", v.Low, err)
		}
		// Parse close price
		c, err := strconv.ParseFloat(v.Close, 64)
		if err != nil {
			return nil, fmt.Errorf("parse close %q: %w", v.Close, err)
		}
		// Parse volume
		vol64, err := strconv.ParseInt(v.Volume, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse volume %q: %w", v.Volume, err)
		}

		// Convert to domain entity
		candles = append(candles, entity.Candle{
			Time:   tm,
			Open:   o,
			High:   h,
			Low:    l,
			Close:  c,
			Volume: vol64,
		})
	}
	return candles, nil
}
