package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"stock_backend/internal/feature/watchlist/domain/entity"
	"stock_backend/internal/feature/watchlist/transport/handler"
	"stock_backend/internal/feature/watchlist/usecase"
	jwtmw "stock_backend/internal/platform/jwt"
)

// mockWatchlistUsecase はWatchlistUsecaseインターフェースのモック実装です。
type mockWatchlistUsecase struct {
	ListUserSymbolsFunc func(ctx context.Context, userID uint) ([]entity.UserSymbol, error)
	AddSymbolFunc       func(ctx context.Context, userID uint, symbolCode string) error
	RemoveSymbolFunc    func(ctx context.Context, userID uint, symbolCode string) error
	ReorderSymbolsFunc  func(ctx context.Context, userID uint, codeOrder []string) error
}

func (m *mockWatchlistUsecase) ListUserSymbols(ctx context.Context, userID uint) ([]entity.UserSymbol, error) {
	if m.ListUserSymbolsFunc != nil {
		return m.ListUserSymbolsFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockWatchlistUsecase) AddSymbol(ctx context.Context, userID uint, symbolCode string) error {
	if m.AddSymbolFunc != nil {
		return m.AddSymbolFunc(ctx, userID, symbolCode)
	}
	return nil
}

func (m *mockWatchlistUsecase) RemoveSymbol(ctx context.Context, userID uint, symbolCode string) error {
	if m.RemoveSymbolFunc != nil {
		return m.RemoveSymbolFunc(ctx, userID, symbolCode)
	}
	return nil
}

func (m *mockWatchlistUsecase) ReorderSymbols(ctx context.Context, userID uint, codeOrder []string) error {
	if m.ReorderSymbolsFunc != nil {
		return m.ReorderSymbolsFunc(ctx, userID, codeOrder)
	}
	return nil
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// setupRouter はテスト用のGinルーターにJWTコンテキストをセットするミドルウェア付きでセットアップします。
func setupRouter(h *handler.WatchlistHandler, userID uint) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(jwtmw.ContextUserID, userID)
		c.Next()
	})
	r.GET("/watchlist", h.List)
	r.POST("/watchlist", h.Add)
	r.PUT("/watchlist/order", h.Reorder)
	r.DELETE("/watchlist/:code", h.Remove)
	return r
}

func TestWatchlistHandler_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockList       func(ctx context.Context, userID uint) ([]entity.UserSymbol, error)
		expectedStatus int
		expectedLen    int
	}{
		{
			name: "success: returns watchlist",
			mockList: func(ctx context.Context, userID uint) ([]entity.UserSymbol, error) {
				return []entity.UserSymbol{
					{ID: 1, UserID: 1, SymbolCode: "AAPL", SortKey: 10},
					{ID: 2, UserID: 1, SymbolCode: "MSFT", SortKey: 20},
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedLen:    2,
		},
		{
			name: "success: empty watchlist",
			mockList: func(ctx context.Context, userID uint) ([]entity.UserSymbol, error) {
				return []entity.UserSymbol{}, nil
			},
			expectedStatus: http.StatusOK,
			expectedLen:    0,
		},
		{
			name: "failure: usecase error",
			mockList: func(ctx context.Context, userID uint) ([]entity.UserSymbol, error) {
				return nil, errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUC := &mockWatchlistUsecase{ListUserSymbolsFunc: tt.mockList}
			h := handler.NewWatchlistHandler(mockUC)
			router := setupRouter(h, 1)

			req, _ := http.NewRequest(http.MethodGet, "/watchlist", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusOK {
				var body []map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &body)
				require.NoError(t, err)
				assert.Len(t, body, tt.expectedLen)
			}
		})
	}
}

func TestWatchlistHandler_Add(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		body           gin.H
		mockAdd        func(ctx context.Context, userID uint, symbolCode string) error
		expectedStatus int
	}{
		{
			name:           "success: add symbol",
			body:           gin.H{"symbol_code": "TSLA"},
			mockAdd:        func(ctx context.Context, userID uint, symbolCode string) error { return nil },
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "failure: invalid request body",
			body:           gin.H{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "failure: duplicate symbol",
			body: gin.H{"symbol_code": "AAPL"},
			mockAdd: func(ctx context.Context, userID uint, symbolCode string) error {
				return usecase.ErrSymbolAlreadyExists
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "failure: usecase error",
			body: gin.H{"symbol_code": "AAPL"},
			mockAdd: func(ctx context.Context, userID uint, symbolCode string) error {
				return errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUC := &mockWatchlistUsecase{AddSymbolFunc: tt.mockAdd}
			h := handler.NewWatchlistHandler(mockUC)
			router := setupRouter(h, 1)

			bodyBytes, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/watchlist", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestWatchlistHandler_Remove(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		code           string
		mockRemove     func(ctx context.Context, userID uint, symbolCode string) error
		expectedStatus int
	}{
		{
			name:           "success: remove symbol",
			code:           "AAPL",
			mockRemove:     func(ctx context.Context, userID uint, symbolCode string) error { return nil },
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "failure: symbol not found",
			code: "UNKNOWN",
			mockRemove: func(ctx context.Context, userID uint, symbolCode string) error {
				return usecase.ErrSymbolNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUC := &mockWatchlistUsecase{RemoveSymbolFunc: tt.mockRemove}
			h := handler.NewWatchlistHandler(mockUC)
			router := setupRouter(h, 1)

			req, _ := http.NewRequest(http.MethodDelete, "/watchlist/"+tt.code, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestWatchlistHandler_Reorder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		body           gin.H
		mockReorder    func(ctx context.Context, userID uint, codeOrder []string) error
		expectedStatus int
	}{
		{
			name: "success: reorder symbols",
			body: gin.H{"symbol_codes": []string{"MSFT", "AAPL", "GOOGL"}},
			mockReorder: func(ctx context.Context, userID uint, codeOrder []string) error {
				return nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "failure: invalid request body",
			body:           gin.H{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "failure: usecase error",
			body: gin.H{"symbol_codes": []string{"AAPL"}},
			mockReorder: func(ctx context.Context, userID uint, codeOrder []string) error {
				return errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUC := &mockWatchlistUsecase{ReorderSymbolsFunc: tt.mockReorder}
			h := handler.NewWatchlistHandler(mockUC)
			router := setupRouter(h, 1)

			bodyBytes, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPut, "/watchlist/order", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
