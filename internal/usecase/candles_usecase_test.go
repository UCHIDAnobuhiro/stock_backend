package usecase

import (
	"context"
	"errors"
	"reflect"
	"stock_backend/internal/domain/entity"
	"testing"
	"time"
)

// sentinel error（モックと期待値で共有）
var ErrDB = errors.New("database error")

// mockCandleRepository は repository.CandleRepository インターフェースのモック実装です。
type mockCandleRepository struct {
	FindFunc        func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
	UpsertBatchFunc func(ctx context.Context, candles []entity.Candle) error
	FindCalls       int
}

func (m *mockCandleRepository) Find(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	m.FindCalls++
	if m.FindFunc != nil {
		return m.FindFunc(ctx, symbol, interval, outputsize)
	}
	return nil, errors.New("FindFunc is not implemented")
}

func (m *mockCandleRepository) UpsertBatch(ctx context.Context, candles []entity.Candle) error {
	if m.UpsertBatchFunc != nil {
		return m.UpsertBatchFunc(ctx, candles)
	}
	return errors.New("UpsertBatchFunc is not implemented")
}

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
		expectedInterval   string // モックに渡されることを期待するinterval
		expectedOutputsize int    // モックに渡されることを期待するoutputsize
	}{
		{
			name:            "正常系: パラメータを全て指定",
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
			name:            "正常系: intervalが空文字の場合、デフォルト値が使われる",
			inputSymbol:     "GOOG",
			inputInterval:   "",
			inputOutputsize: 100,
			mockFindFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return expectedCandles, nil
			},
			expectedCandles:    expectedCandles,
			expectedErr:        nil,
			expectedInterval:   DefaultInterval, // 例: "1day"
			expectedOutputsize: 100,
		},
		{
			name:            "正常系: outputsizeが0の場合、デフォルト値が使われる",
			inputSymbol:     "MSFT",
			inputInterval:   "1month",
			inputOutputsize: 0,
			mockFindFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return expectedCandles, nil
			},
			expectedCandles:    expectedCandles,
			expectedErr:        nil,
			expectedInterval:   "1month",
			expectedOutputsize: DefaultOutputSize, // 例: 200
		},
		{
			name:            "正常系: outputsizeが最大値を超える場合、デフォルト値が使われる",
			inputSymbol:     "TSLA",
			inputInterval:   "1day",
			inputOutputsize: MaxOutputSize + 1,
			mockFindFunc: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return expectedCandles, nil
			},
			expectedCandles:    expectedCandles,
			expectedErr:        nil,
			expectedInterval:   "1day",
			expectedOutputsize: DefaultOutputSize, // 例: 200
		},
		{
			name:            "異常系: リポジトリがエラーを返す",
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
					// ユースケースが正しいパラメータでリポジトリを呼び出したか検証
					if symbol != tc.inputSymbol || interval != tc.expectedInterval || outputsize != tc.expectedOutputsize {
						t.Errorf("Findに予期しないパラメータが渡されました。got symbol=%s, interval=%s, outputsize=%d, want symbol=%s, interval=%s, outputsize=%d",
							symbol, interval, outputsize, tc.inputSymbol, tc.expectedInterval, tc.expectedOutputsize)
					}
					return tc.mockFindFunc(ctx, symbol, interval, outputsize)
				},
			}
			uc := NewCandlesUsecase(mockRepo)

			candles, err := uc.GetCandles(ctx, tc.inputSymbol, tc.inputInterval, tc.inputOutputsize)

			// エラー判定（sentinelで簡潔に）
			if tc.expectedErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected %v, got %v", tc.expectedErr, err)
			}

			// 結果比較
			if !reflect.DeepEqual(candles, tc.expectedCandles) {
				t.Errorf("期待した結果と異なる結果が返されました。got %v, want %v", candles, tc.expectedCandles)
			}

			// 呼び出し回数
			if mockRepo.FindCalls != 1 {
				t.Errorf("Findが%d回呼ばれました（期待: 1）", mockRepo.FindCalls)
			}
		})
	}
}
