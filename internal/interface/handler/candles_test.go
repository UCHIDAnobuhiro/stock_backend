package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"stock_backend/internal/domain/entity"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// mockCandlesUsecase は candlesUsecase インターフェースのモック実装です。
type mockCandlesUsecase struct {
	GetCandlesFunc func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
}

func (m *mockCandlesUsecase) GetCandles(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	return m.GetCandlesFunc(ctx, symbol, interval, outputsize)
}

func TestCandlesHandler_GetCandlesHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// テスト用の固定時刻
	testTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		url            string
		mockGetCandles func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
		expectedStatus int
		expectedBody   string // JSON文字列で比較
	}{
		{
			name: "成功: 全てのパラメータを指定",
			url:  "/candles/7203.T?interval=1day&outputsize=10",
			mockGetCandles: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				assert.Equal(t, "7203.T", symbol)
				assert.Equal(t, "1day", interval)
				assert.Equal(t, 10, outputsize)
				return []entity.Candle{
					{Time: testTime, Open: 100, High: 110, Low: 90, Close: 105, Volume: 1000},
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"time":"2023-01-01","open":100,"high":110,"low":90,"close":105,"volume":1000}]`,
		},
		{
			name: "成功: パラメータがデフォルト値",
			url:  "/candles/7203.T",
			mockGetCandles: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				assert.Equal(t, "7203.T", symbol)
				assert.Equal(t, "1day", interval) // デフォルト値
				assert.Equal(t, 200, outputsize)  // デフォルト値
				return []entity.Candle{}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[]`,
		},
		{
			name: "失敗: Usecaseがエラーを返す",
			url:  "/candles/9999.T",
			mockGetCandles: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return nil, errors.New("internal server error")
			},
			expectedStatus: http.StatusBadGateway,
			expectedBody:   `{"error":"internal server error"}`,
		},
		{
			name: "準正常: outputsizeが不正な文字列の場合、デフォルト値が使われる",
			url:  "/candles/7203.T?outputsize=invalid",
			mockGetCandles: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				// ハンドラは strconv.Atoi("invalid") の結果である 0 をUsecaseに渡すのが責務。
				// デフォルト値への変換はUsecase層で行われる。
				assert.Equal(t, 0, outputsize)
				return []entity.Candle{}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックUsecaseをインスタンス化
			mockUC := &mockCandlesUsecase{
				GetCandlesFunc: tt.mockGetCandles,
			}

			handler := NewCandlesHandler(mockUC)

			router := gin.New()
			router.GET("/candles/:code", handler.GetCandlesHandler)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, tt.url, io.NopCloser(bytes.NewReader(nil)))

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}
