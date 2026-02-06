package handler_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"stock_backend/internal/feature/candles/domain/entity"
	"stock_backend/internal/feature/candles/transport/handler"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// mockCandlesUsecase はcandlesUsecaseインターフェースのモック実装です。
type mockCandlesUsecase struct {
	GetCandlesFunc func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
}

func (m *mockCandlesUsecase) GetCandles(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	return m.GetCandlesFunc(ctx, symbol, interval, outputsize)
}

// TestCandlesHandler_GetCandlesHandler はGetCandlesHandlerのHTTPリクエスト/レスポンス処理をテストします。
func TestCandlesHandler_GetCandlesHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// テスト用の固定時刻
	testTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		url            string
		mockGetCandles func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
		expectedStatus int
		expectedBody   string // JSON文字列として比較
	}{
		{
			name: "success: all parameters specified",
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
			name: "success: default parameter values",
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
			name: "error: usecase returns error",
			url:  "/candles/9999.T",
			mockGetCandles: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				return nil, errors.New("internal server error")
			},
			expectedStatus: http.StatusBadGateway,
			expectedBody:   `{"error":"internal server error"}`,
		},
		{
			name: "edge case: invalid outputsize string uses default value",
			url:  "/candles/7203.T?outputsize=invalid",
			mockGetCandles: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
				// ハンドラーは0（strconv.Atoi("invalid")の結果）をusecaseに渡す。
				// デフォルト値への変換はusecaseレイヤーで処理される。
				assert.Equal(t, 0, outputsize)
				return []entity.Candle{}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックusecaseのインスタンスを生成
			mockUC := &mockCandlesUsecase{
				GetCandlesFunc: tt.mockGetCandles,
			}

			h := handler.NewCandlesHandler(mockUC)

			router := gin.New()
			router.GET("/candles/:code", h.GetCandlesHandler)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, tt.url, io.NopCloser(bytes.NewReader(nil)))

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}
