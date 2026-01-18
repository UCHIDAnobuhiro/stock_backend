package usecase

import (
	"context"
	"errors"
	"stock_backend/internal/feature/candles/domain/entity"
	"testing"
	"time"
)

var ErrMarketAPI = errors.New("market API error")

// mockMarketRepository is a mock implementation of the MarketRepository interface.
type mockMarketRepository struct {
	GetTimeSeriesFunc  func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
	GetTimeSeriesCalls int
}

func (m *mockMarketRepository) GetTimeSeries(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	m.GetTimeSeriesCalls++
	if m.GetTimeSeriesFunc != nil {
		return m.GetTimeSeriesFunc(ctx, symbol, interval, outputsize)
	}
	return nil, errors.New("GetTimeSeriesFunc is not implemented")
}

// mockRateLimiter is a mock implementation of the RateLimiterInterface.
type mockRateLimiter struct {
	WaitIfNeededCalls int
}

func (m *mockRateLimiter) WaitIfNeeded() {
	m.WaitIfNeededCalls++
	// For testing purposes, return immediately without waiting
}

func TestIngestUsecase_ingestOne(t *testing.T) {
	ctx := context.Background()
	testTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	mockCandles := []entity.Candle{
		{Time: testTime, Open: 100, High: 110, Low: 90, Close: 105},
		{Time: testTime.AddDate(0, 0, -1), Open: 95, High: 105, Low: 85, Close: 100},
	}

	testCases := []struct {
		name                 string
		inputSymbol          string
		inputInterval        string
		inputOutputsize      int
		mockGetTimeSeriesFunc func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
		mockUpsertBatchFunc  func(ctx context.Context, candles []entity.Candle) error
		expectedErr          error
		verifyCandles        func(t *testing.T, candles []entity.Candle)
	}{
		{
			name:            "success: data fetch and save succeed",
			inputSymbol:     "AAPL",
			inputInterval:   "1day",
			inputOutputsize: 200,
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				if symbol != "AAPL" || interval != "1day" || outputsize != 200 {
					t.Errorf("GetTimeSeries called with unexpected params: got symbol=%s, interval=%s, outputsize=%d", symbol, interval, outputsize)
				}
				return mockCandles, nil
			},
			mockUpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
				return nil
			},
			expectedErr: nil,
			verifyCandles: func(t *testing.T, candles []entity.Candle) {
				if len(candles) != 2 {
					t.Errorf("candles count mismatch: got %d, want 2", len(candles))
				}
				for _, c := range candles {
					if c.Symbol != "AAPL" {
						t.Errorf("candle Symbol not set: got %s, want AAPL", c.Symbol)
					}
					if c.Interval != "1day" {
						t.Errorf("candle Interval not set: got %s, want 1day", c.Interval)
					}
				}
			},
		},
		{
			name:            "error: MarketRepository returns error",
			inputSymbol:     "GOOG",
			inputInterval:   "1week",
			inputOutputsize: 100,
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return nil, ErrMarketAPI
			},
			mockUpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
				t.Error("UpsertBatch should not be called")
				return nil
			},
			expectedErr: ErrMarketAPI,
		},
		{
			name:            "error: CandleRepository returns error",
			inputSymbol:     "MSFT",
			inputInterval:   "1month",
			inputOutputsize: 50,
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return mockCandles, nil
			},
			mockUpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
				return ErrDB
			},
			expectedErr: ErrDB,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedCandles []entity.Candle
			mockMarket := &mockMarketRepository{
				GetTimeSeriesFunc: tc.mockGetTimeSeriesFunc,
			}
			mockCandle := &mockCandleRepository{
				UpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
					capturedCandles = candles
					return tc.mockUpsertBatchFunc(ctx, candles)
				},
			}
			mockRL := &mockRateLimiter{}

			uc := NewIngestUsecase(mockMarket, mockCandle, mockRL)
			err := uc.ingestOne(ctx, tc.inputSymbol, tc.inputInterval, tc.inputOutputsize)

			if tc.expectedErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected %v, got %v", tc.expectedErr, err)
			}

			if tc.verifyCandles != nil && capturedCandles != nil {
				tc.verifyCandles(t, capturedCandles)
			}

			if mockMarket.GetTimeSeriesCalls != 1 {
				t.Errorf("GetTimeSeries was called %d times, expected 1", mockMarket.GetTimeSeriesCalls)
			}
		})
	}
}

func TestIngestUsecase_IngestAll(t *testing.T) {
	ctx := context.Background()
	testTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	mockCandles := []entity.Candle{
		{Time: testTime, Open: 100, High: 110, Low: 90, Close: 105},
	}

	testCases := []struct {
		name                    string
		inputSymbols            []string
		mockGetTimeSeriesFunc   func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
		mockUpsertBatchFunc     func(ctx context.Context, candles []entity.Candle) error
		expectedErr             error
		expectedGetTimeSeriesCalls int
	}{
		{
			name:         "success: fetch all symbols and intervals",
			inputSymbols: []string{"AAPL", "GOOG"},
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return mockCandles, nil
			},
			mockUpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
				return nil
			},
			expectedErr: nil,
			// 2 symbols × 3 intervals (1day, 1week, 1month) = 6 calls
			expectedGetTimeSeriesCalls: 6,
		},
		{
			name:         "success: single symbol fetch succeeds",
			inputSymbols: []string{"TSLA"},
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return mockCandles, nil
			},
			mockUpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
				return nil
			},
			expectedErr: nil,
			// 1 symbol × 3 intervals = 3 calls
			expectedGetTimeSeriesCalls: 3,
		},
		{
			name:         "success: empty symbol list",
			inputSymbols: []string{},
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				t.Error("GetTimeSeries should not be called")
				return nil, errors.New("should not be called")
			},
			mockUpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
				t.Error("UpsertBatch should not be called")
				return nil
			},
			expectedErr:                nil,
			expectedGetTimeSeriesCalls: 0,
		},
		{
			name:         "success: continues processing even when some symbols fail",
			inputSymbols: []string{"AAPL", "INVALID", "GOOG"},
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				if symbol == "INVALID" {
					return nil, ErrMarketAPI
				}
				return mockCandles, nil
			},
			mockUpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
				return nil
			},
			expectedErr: nil, // IngestAll continues without returning error
			// 3 symbols × 3 intervals = 9 calls (even with errors, all calls are attempted)
			expectedGetTimeSeriesCalls: 9,
		},
		{
			name:         "success: continues processing even when UpsertBatch fails",
			inputSymbols: []string{"AAPL", "GOOG"},
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return mockCandles, nil
			},
			mockUpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
				if candles[0].Symbol == "AAPL" {
					return ErrDB
				}
				return nil
			},
			expectedErr: nil, // IngestAll continues without returning error
			// 2 symbols × 3 intervals = 6 calls
			expectedGetTimeSeriesCalls: 6,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockMarket := &mockMarketRepository{
				GetTimeSeriesFunc: tc.mockGetTimeSeriesFunc,
			}
			mockCandle := &mockCandleRepository{
				UpsertBatchFunc: tc.mockUpsertBatchFunc,
			}
			mockRL := &mockRateLimiter{}

			uc := NewIngestUsecase(mockMarket, mockCandle, mockRL)
			err := uc.IngestAll(ctx, tc.inputSymbols)

			if tc.expectedErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected %v, got %v", tc.expectedErr, err)
			}

			if mockMarket.GetTimeSeriesCalls != tc.expectedGetTimeSeriesCalls {
				t.Errorf("GetTimeSeries was called %d times, expected %d", mockMarket.GetTimeSeriesCalls, tc.expectedGetTimeSeriesCalls)
			}
		})
	}
}

func TestIngestUsecase_IngestAll_Intervals(t *testing.T) {
	ctx := context.Background()
	testTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	mockCandles := []entity.Candle{
		{Time: testTime, Open: 100, High: 110, Low: 90, Close: 105},
	}

	calledIntervals := []string{}

	mockMarket := &mockMarketRepository{
		GetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
			calledIntervals = append(calledIntervals, interval)
			return mockCandles, nil
		},
	}
	mockCandle := &mockCandleRepository{
		UpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
			return nil
		},
	}
	mockRL := &mockRateLimiter{}

	uc := NewIngestUsecase(mockMarket, mockCandle, mockRL)
	err := uc.IngestAll(ctx, []string{"AAPL"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedIntervals := []string{"1day", "1week", "1month"}
	if len(calledIntervals) != len(expectedIntervals) {
		t.Fatalf("intervals count mismatch: got %d, want %d", len(calledIntervals), len(expectedIntervals))
	}

	for i, expected := range expectedIntervals {
		if calledIntervals[i] != expected {
			t.Errorf("interval[%d] mismatch: got %s, want %s", i, calledIntervals[i], expected)
		}
	}
}
