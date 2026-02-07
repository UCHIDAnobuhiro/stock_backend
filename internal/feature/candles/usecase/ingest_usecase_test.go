package usecase

import (
	"context"
	"errors"
	"stock_backend/internal/feature/candles/domain/entity"
	"testing"
	"time"
)

// ErrDB はデータベースのセンチネルエラーです。
var ErrDB = errors.New("database error")

// ErrMarketAPI はマーケットAPIのセンチネルエラーです。
var ErrMarketAPI = errors.New("market API error")

// mockCandleRepository はCandleRepositoryインターフェースのモック実装です。
type mockCandleRepository struct {
	FindFunc        func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
	UpsertBatchFunc func(ctx context.Context, candles []entity.Candle) error
	FindCalls       int
}

// Find はFindFuncが設定されていればそれを呼び出し、呼び出し回数を記録します。
func (m *mockCandleRepository) Find(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	m.FindCalls++
	if m.FindFunc != nil {
		return m.FindFunc(ctx, symbol, interval, outputsize)
	}
	return nil, errors.New("FindFunc is not implemented")
}

// UpsertBatch はUpsertBatchFuncが設定されていればそれを呼び出します。
func (m *mockCandleRepository) UpsertBatch(ctx context.Context, candles []entity.Candle) error {
	if m.UpsertBatchFunc != nil {
		return m.UpsertBatchFunc(ctx, candles)
	}
	return errors.New("UpsertBatchFunc is not implemented")
}

// mockMarketRepository はMarketRepositoryインターフェースのモック実装です。
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

// mockSymbolRepository はSymbolRepositoryインターフェースのモック実装です。
type mockSymbolRepository struct {
	ListActiveCodesFunc  func(ctx context.Context) ([]string, error)
	ListActiveCodesCalls int
}

func (m *mockSymbolRepository) ListActiveCodes(ctx context.Context) ([]string, error) {
	m.ListActiveCodesCalls++
	if m.ListActiveCodesFunc != nil {
		return m.ListActiveCodesFunc(ctx)
	}
	return nil, errors.New("ListActiveCodesFunc is not implemented")
}

// mockRateLimiter はRateLimiterInterfaceのモック実装です。
type mockRateLimiter struct {
	WaitIfNeededCalls int
}

func (m *mockRateLimiter) WaitIfNeeded() {
	m.WaitIfNeededCalls++
	// テスト用に待機せず即座にリターン
}

// TestIngestUsecase_ingestOne はingestOneメソッドのデータ取得・保存処理をテストします。
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
			mockSymbol := &mockSymbolRepository{}

			uc := NewIngestUsecase(mockMarket, mockCandle, mockSymbol, mockRL)
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

// TestIngestUsecase_IngestAll はIngestAllメソッドの全銘柄・全インターバル処理をテストします。
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
			// 2銘柄 × 3インターバル（1day, 1week, 1month）= 6回呼び出し
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
			// 1銘柄 × 3インターバル = 3回呼び出し
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
			expectedErr: nil, // IngestAllはエラーを返さず処理を続行
			// 3銘柄 × 3インターバル = 9回呼び出し（エラーが発生しても全呼び出しが試行される）
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
			expectedErr: nil, // IngestAllはエラーを返さず処理を続行
			// 2銘柄 × 3インターバル = 6回呼び出し
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
			mockSymbol := &mockSymbolRepository{
				ListActiveCodesFunc: func(ctx context.Context) ([]string, error) {
					return tc.inputSymbols, nil
				},
			}
			mockRL := &mockRateLimiter{}

			uc := NewIngestUsecase(mockMarket, mockCandle, mockSymbol, mockRL)
			err := uc.IngestAll(ctx)

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

// TestIngestUsecase_IngestAll_Intervals はIngestAllが正しいインターバル順序で呼び出すことをテストします。
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
	mockSymbol := &mockSymbolRepository{
		ListActiveCodesFunc: func(ctx context.Context) ([]string, error) {
			return []string{"AAPL"}, nil
		},
	}
	mockRL := &mockRateLimiter{}

	uc := NewIngestUsecase(mockMarket, mockCandle, mockSymbol, mockRL)
	err := uc.IngestAll(ctx)

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
