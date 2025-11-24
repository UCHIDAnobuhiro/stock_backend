package usecase

import (
	"context"
	"errors"
	"stock_backend/internal/feature/candles/domain/entity"
	"testing"
	"time"
)

var ErrMarketAPI = errors.New("market API error")

// mockMarketRepository は repository.MarketRepository インターフェースのモック実装です。
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

// mockRateLimiter は ratelimiter.RateLimiterInterface のモック実装です。
type mockRateLimiter struct {
	WaitIfNeededCalls int
}

func (m *mockRateLimiter) WaitIfNeeded() {
	m.WaitIfNeededCalls++
	// テスト用なので何もせず即座にリターン
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
			name:            "正常系: データの取得と保存が成功",
			inputSymbol:     "AAPL",
			inputInterval:   "1day",
			inputOutputsize: 200,
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				if symbol != "AAPL" || interval != "1day" || outputsize != 200 {
					t.Errorf("GetTimeSeriesに予期しないパラメータが渡されました。got symbol=%s, interval=%s, outputsize=%d", symbol, interval, outputsize)
				}
				return mockCandles, nil
			},
			mockUpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
				return nil
			},
			expectedErr: nil,
			verifyCandles: func(t *testing.T, candles []entity.Candle) {
				if len(candles) != 2 {
					t.Errorf("candlesの数が期待と異なります。got %d, want 2", len(candles))
				}
				for _, c := range candles {
					if c.Symbol != "AAPL" {
						t.Errorf("candleのSymbolが設定されていません。got %s, want AAPL", c.Symbol)
					}
					if c.Interval != "1day" {
						t.Errorf("candleのIntervalが設定されていません。got %s, want 1day", c.Interval)
					}
				}
			},
		},
		{
			name:            "異常系: MarketRepositoryがエラーを返す",
			inputSymbol:     "GOOG",
			inputInterval:   "1week",
			inputOutputsize: 100,
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return nil, ErrMarketAPI
			},
			mockUpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
				t.Error("UpsertBatchが呼ばれるべきではありません")
				return nil
			},
			expectedErr: ErrMarketAPI,
		},
		{
			name:            "異常系: CandleRepositoryがエラーを返す",
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
				t.Errorf("GetTimeSeriesが%d回呼ばれました（期待: 1）", mockMarket.GetTimeSeriesCalls)
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
			name:         "正常系: 複数銘柄×複数時間足の全取得が成功",
			inputSymbols: []string{"AAPL", "GOOG"},
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return mockCandles, nil
			},
			mockUpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
				return nil
			},
			expectedErr: nil,
			// 2銘柄 × 3時間足(1day, 1week, 1month) = 6回
			expectedGetTimeSeriesCalls: 6,
		},
		{
			name:         "正常系: 単一銘柄の全取得が成功",
			inputSymbols: []string{"TSLA"},
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return mockCandles, nil
			},
			mockUpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
				return nil
			},
			expectedErr: nil,
			// 1銘柄 × 3時間足 = 3回
			expectedGetTimeSeriesCalls: 3,
		},
		{
			name:         "正常系: 空の銘柄リスト",
			inputSymbols: []string{},
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				t.Error("GetTimeSeriesが呼ばれるべきではありません")
				return nil, errors.New("should not be called")
			},
			mockUpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
				t.Error("UpsertBatchが呼ばれるべきではありません")
				return nil
			},
			expectedErr:                nil,
			expectedGetTimeSeriesCalls: 0,
		},
		{
			name:         "正常系: 一部の銘柄でエラーが発生しても処理は継続",
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
			expectedErr: nil, // IngestAllはエラーを返さずに続行する
			// 3銘柄 × 3時間足 = 9回（エラーが発生しても呼び出しは行われる）
			expectedGetTimeSeriesCalls: 9,
		},
		{
			name:         "正常系: UpsertBatchでエラーが発生しても処理は継続",
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
			expectedErr: nil, // IngestAllはエラーを返さずに続行する
			// 2銘柄 × 3時間足 = 6回
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
				t.Errorf("GetTimeSeriesが%d回呼ばれました（期待: %d）", mockMarket.GetTimeSeriesCalls, tc.expectedGetTimeSeriesCalls)
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
		t.Fatalf("呼び出されたintervalsの数が異なります。got %d, want %d", len(calledIntervals), len(expectedIntervals))
	}

	for i, expected := range expectedIntervals {
		if calledIntervals[i] != expected {
			t.Errorf("interval[%d]が異なります。got %s, want %s", i, calledIntervals[i], expected)
		}
	}
}
