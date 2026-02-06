package usecase_test

import (
	"context"
	"errors"
	"reflect"
	"stock_backend/internal/feature/candles/domain/entity"
	"stock_backend/internal/feature/candles/usecase"
	"testing"
	"time"
)

// ErrDB はモックと期待値の間で共有されるセンチネルエラーです。
var ErrDB = errors.New("database error")

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

// TestCandlesUsecase_GetCandles はGetCandlesメソッドのパラメータ処理とリポジトリ呼び出しをテストします。
func TestCandlesUsecase_GetCandles(t *testing.T) {
	ctx := context.Background()
	expectedCandles := []entity.Candle{
		{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 110, Low: 90, Close: 105},
	}

	testCases := []struct {
		name               string
		inputSymbol        string
		inputInterval      string
		inputOutputsize    int
		mockFindFunc       func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
		expectedCandles    []entity.Candle
		expectedErr        error
		expectedInterval   string // モックに渡されるべきインターバル
		expectedOutputsize int    // モックに渡されるべきoutputsize
	}{
		{
			name:            "success: all parameters specified",
			inputSymbol:     "AAPL",
			inputInterval:   "1week",
			inputOutputsize: 50,
			mockFindFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return expectedCandles, nil
			},
			expectedCandles:    expectedCandles,
			expectedErr:        nil,
			expectedInterval:   "1week",
			expectedOutputsize: 50,
		},
		{
			name:            "success: default value used when interval is empty",
			inputSymbol:     "GOOG",
			inputInterval:   "",
			inputOutputsize: 100,
			mockFindFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return expectedCandles, nil
			},
			expectedCandles:    expectedCandles,
			expectedErr:        nil,
			expectedInterval:   "1day",
			expectedOutputsize: 100,
		},
		{
			name:            "success: default value used when outputsize is 0",
			inputSymbol:     "MSFT",
			inputInterval:   "1month",
			inputOutputsize: 0,
			mockFindFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return expectedCandles, nil
			},
			expectedCandles:    expectedCandles,
			expectedErr:        nil,
			expectedInterval:   "1month",
			expectedOutputsize: 200,
		},
		{
			name:            "success: default value used when outputsize exceeds max",
			inputSymbol:     "TSLA",
			inputInterval:   "1day",
			inputOutputsize: 5001,
			mockFindFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return expectedCandles, nil
			},
			expectedCandles:    expectedCandles,
			expectedErr:        nil,
			expectedInterval:   "1day",
			expectedOutputsize: 200,
		},
		{
			name:            "error: repository returns error",
			inputSymbol:     "AMZN",
			inputInterval:   "1day",
			inputOutputsize: 10,
			mockFindFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return nil, ErrDB
			},
			expectedCandles:    nil,
			expectedErr:        ErrDB,
			expectedInterval:   "1day",
			expectedOutputsize: 10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockCandleRepository{
				FindFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
					// ユースケースが正しいパラメータでリポジトリを呼び出すことを検証
					if symbol != tc.inputSymbol || interval != tc.expectedInterval || outputsize != tc.expectedOutputsize {
						t.Errorf("Find called with unexpected params: got symbol=%s, interval=%s, outputsize=%d, want symbol=%s, interval=%s, outputsize=%d",
							symbol, interval, outputsize, tc.inputSymbol, tc.expectedInterval, tc.expectedOutputsize)
					}
					return tc.mockFindFunc(ctx, symbol, interval, outputsize)
				},
			}
			uc := usecase.NewCandlesUsecase(mockRepo)

			candles, err := uc.GetCandles(ctx, tc.inputSymbol, tc.inputInterval, tc.inputOutputsize)

			// センチネル比較によるエラー検証
			if tc.expectedErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected %v, got %v", tc.expectedErr, err)
			}

			// 結果の比較
			if !reflect.DeepEqual(candles, tc.expectedCandles) {
				t.Errorf("result mismatch: got %v, want %v", candles, tc.expectedCandles)
			}

			// 呼び出し回数の検証
			if mockRepo.FindCalls != 1 {
				t.Errorf("Find was called %d times, expected 1", mockRepo.FindCalls)
			}
		})
	}
}
