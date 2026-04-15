package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"stock_backend/internal/feature/candles/domain/entity"
)

// ErrDB はデータベースのセンチネルエラーです。
var ErrDB = errors.New("database error")

// ErrMarketAPI はマーケットAPIのセンチネルエラーです。
var ErrMarketAPI = errors.New("market API error")

// mockCandleWriteRepository はCandleWriteRepositoryインターフェースのモック実装です。
type mockCandleWriteRepository struct {
	UpsertBatchFunc func(ctx context.Context, candles []entity.Candle) error
}

// UpsertBatch はUpsertBatchFuncが設定されていればそれを呼び出します。
func (m *mockCandleWriteRepository) UpsertBatch(ctx context.Context, candles []entity.Candle) error {
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

// TestDedupCandles は dedupCandles 関数の重複除去ロジックをテストします。
func TestDedupCandles(t *testing.T) {
	base := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name     string
		input    []entity.Candle
		wantLen  int
		wantKeys []string // "symbol|interval|unix" 形式で期待するキー一覧
	}{
		{
			name: "重複なしの場合は全件返す",
			input: []entity.Candle{
				{Symbol: "AAPL", Interval: "1day", Time: base},
				{Symbol: "AAPL", Interval: "1day", Time: base.AddDate(0, 0, 1)},
			},
			wantLen: 2,
		},
		{
			name: "同一タイムスタンプの重複は1件に絞る",
			input: []entity.Candle{
				{Symbol: "AAPL", Interval: "1day", Time: base},
				{Symbol: "AAPL", Interval: "1day", Time: base},
			},
			wantLen: 1,
		},
		{
			name: "symbolが異なれば別エントリとして扱う",
			input: []entity.Candle{
				{Symbol: "AAPL", Interval: "1day", Time: base},
				{Symbol: "GOOG", Interval: "1day", Time: base},
			},
			wantLen: 2,
		},
		{
			name: "intervalが異なれば別エントリとして扱う",
			input: []entity.Candle{
				{Symbol: "AAPL", Interval: "1day", Time: base},
				{Symbol: "AAPL", Interval: "1week", Time: base},
			},
			wantLen: 2,
		},
		{
			name:    "空スライスは空スライスを返す",
			input:   []entity.Candle{},
			wantLen: 0,
		},
		{
			name: "元スライスを変更しない（backing array 非共有）",
			input: []entity.Candle{
				{Symbol: "AAPL", Interval: "1day", Time: base, Close: 100},
				{Symbol: "AAPL", Interval: "1day", Time: base, Close: 200}, // 重複
				{Symbol: "AAPL", Interval: "1day", Time: base.AddDate(0, 0, 1), Close: 300},
			},
			wantLen: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 元スライスのコピーを保持して変更されていないか確認
			original := make([]entity.Candle, len(tc.input))
			copy(original, tc.input)

			got := dedupCandles(tc.input)

			if len(got) != tc.wantLen {
				t.Errorf("len=%d, want %d", len(got), tc.wantLen)
			}

			// 元スライスが変更されていないことを確認
			for i, c := range tc.input {
				if c != original[i] {
					t.Errorf("input[%d] was modified: got %+v, want %+v", i, c, original[i])
				}
			}

			// 出力に重複がないことを確認
			seen := make(map[string]struct{})
			for _, c := range got {
				key := fmt.Sprintf("%s|%s|%d", c.Symbol, c.Interval, c.Time.Unix())
				if _, exists := seen[key]; exists {
					t.Errorf("duplicate key in output: %s", key)
				}
				seen[key] = struct{}{}
			}
		})
	}
}

// TestIngestUsecase_ingestOne はingestOneメソッドのデータ取得・保存処理をテストします。
func TestIngestUsecase_ingestOne(t *testing.T) {
	ctx := context.Background()
	// 2022-12-31（土）と 2023-01-01（日）は同一 ISO 週（2022-W52）かつ異なる月
	testTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	mockDailyCandles := []entity.Candle{
		{Time: testTime, Open: 100, High: 110, Low: 90, Close: 105},
		{Time: testTime.AddDate(0, 0, -1), Open: 95, High: 105, Low: 85, Close: 100},
	}

	testCases := []struct {
		name                  string
		inputSymbol           string
		inputOutputsize       int
		mockGetTimeSeriesFunc func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
		mockUpsertBatchFunc   func(ctx context.Context, candles []entity.Candle) error
		expectedErr           error
		verifyCandles         func(t *testing.T, candles []entity.Candle)
	}{
		{
			name:            "success: data fetch and save succeed",
			inputSymbol:     "AAPL",
			inputOutputsize: 5000,
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				if symbol != "AAPL" {
					t.Errorf("GetTimeSeries called with unexpected symbol: got %s, want AAPL", symbol)
				}
				if interval != "1day" {
					t.Errorf("GetTimeSeries must always be called with interval=1day, got %s", interval)
				}
				return mockDailyCandles, nil
			},
			mockUpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
				return nil
			},
			expectedErr: nil,
			// 2日足
			// + 0週足（2022-12-31 は土曜 = 不完全な先頭バケットを除外）
			// + 1月足（2022-12 は 31日開始 = 不完全で除外、2023-01 は 1日開始で保持）
			// = 3件
			verifyCandles: func(t *testing.T, candles []entity.Candle) {
				counts := map[string]int{}
				for _, c := range candles {
					if c.Symbol != "AAPL" {
						t.Errorf("candle Symbol not set: got %s, want AAPL", c.Symbol)
					}
					counts[c.Interval]++
				}
				if counts["1day"] != 2 {
					t.Errorf("1day candle count: got %d, want 2", counts["1day"])
				}
				if counts["1week"] != 0 {
					t.Errorf("1week candle count: got %d, want 0 (incomplete first bucket dropped)", counts["1week"])
				}
				if counts["1month"] != 1 {
					t.Errorf("1month candle count: got %d, want 1 (incomplete December bucket dropped)", counts["1month"])
				}
			},
		},
		{
			name:            "error: MarketRepository returns error",
			inputSymbol:     "GOOG",
			inputOutputsize: 5000,
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
			inputOutputsize: 5000,
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return mockDailyCandles, nil
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
			mockCandle := &mockCandleWriteRepository{
				UpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
					capturedCandles = candles
					return tc.mockUpsertBatchFunc(ctx, candles)
				},
			}
			mockRL := &mockRateLimiter{}
			mockSymbol := &mockSymbolRepository{}

			uc := NewIngestUsecase(mockMarket, mockCandle, mockSymbol, mockRL)
			err := uc.ingestOne(ctx, tc.inputSymbol, tc.inputOutputsize)

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

// TestIngestUsecase_IngestAll はIngestAllメソッドの全銘柄処理をテストします。
func TestIngestUsecase_IngestAll(t *testing.T) {
	ctx := context.Background()
	testTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	mockCandles := []entity.Candle{
		{Time: testTime, Open: 100, High: 110, Low: 90, Close: 105},
	}

	testCases := []struct {
		name                       string
		inputSymbols               []string
		mockGetTimeSeriesFunc      func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
		mockUpsertBatchFunc        func(ctx context.Context, candles []entity.Candle) error
		expectedErr                error
		expectedGetTimeSeriesCalls int
		verifyCalledIntervals      func(t *testing.T, intervals []string)
	}{
		{
			name:         "success: fetch all symbols",
			inputSymbols: []string{"AAPL", "GOOG"},
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return mockCandles, nil
			},
			mockUpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
				return nil
			},
			expectedErr: nil,
			// 2銘柄 × 1回（日足のみ取得）= 2回呼び出し
			expectedGetTimeSeriesCalls: 2,
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
			// 1銘柄 × 1回 = 1回呼び出し
			expectedGetTimeSeriesCalls: 1,
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
			// 3銘柄 × 1回 = 3回呼び出し（エラーが発生しても全銘柄が試行される）
			expectedGetTimeSeriesCalls: 3,
		},
		{
			name:         "success: continues processing even when UpsertBatch fails",
			inputSymbols: []string{"AAPL", "GOOG"},
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return mockCandles, nil
			},
			mockUpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
				if len(candles) > 0 && candles[0].Symbol == "AAPL" {
					return ErrDB
				}
				return nil
			},
			expectedErr: nil, // IngestAllはエラーを返さず処理を続行
			// 2銘柄 × 1回 = 2回呼び出し
			expectedGetTimeSeriesCalls: 2,
		},
		{
			name:         "success: only fetches 1day interval from API",
			inputSymbols: []string{"AAPL", "GOOG"},
			mockGetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return mockCandles, nil
			},
			mockUpsertBatchFunc: func(ctx context.Context, candles []entity.Candle) error {
				return nil
			},
			expectedErr:                nil,
			expectedGetTimeSeriesCalls: 2,
			verifyCalledIntervals: func(t *testing.T, intervals []string) {
				for i, interval := range intervals {
					if interval != "1day" {
						t.Errorf("call[%d]: interval=%s, want 1day", i, interval)
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var calledIntervals []string
			mockMarket := &mockMarketRepository{
				GetTimeSeriesFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
					calledIntervals = append(calledIntervals, interval)
					return tc.mockGetTimeSeriesFunc(ctx, symbol, interval, outputsize)
				},
			}
			mockCandle := &mockCandleWriteRepository{
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

			if tc.verifyCalledIntervals != nil {
				tc.verifyCalledIntervals(t, calledIntervals)
			}
		})
	}
}
