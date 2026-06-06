package watchlisthttp_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"stock_backend/internal/feature/watchlist"
	"stock_backend/internal/feature/watchlist/watchlisthttp"
	jwtmw "stock_backend/internal/platform/jwt"
)

const testUserID int64 = 1

// mockUsecase は Usecase インターフェースのモック実装です。
type mockUsecase struct {
	ListUserSymbolsFunc func(ctx context.Context, userID int64) ([]watchlist.UserSymbol, error)
	AddSymbolFunc       func(ctx context.Context, userID int64, symbolCode string) error
	RemoveSymbolFunc    func(ctx context.Context, userID int64, symbolCode string) error
	ReorderSymbolsFunc  func(ctx context.Context, userID int64, orderedCodes []string) error
}

func (m *mockUsecase) ListUserSymbols(ctx context.Context, userID int64) ([]watchlist.UserSymbol, error) {
	if m.ListUserSymbolsFunc != nil {
		return m.ListUserSymbolsFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockUsecase) AddSymbol(ctx context.Context, userID int64, symbolCode string) error {
	if m.AddSymbolFunc != nil {
		return m.AddSymbolFunc(ctx, userID, symbolCode)
	}
	return nil
}

func (m *mockUsecase) RemoveSymbol(ctx context.Context, userID int64, symbolCode string) error {
	if m.RemoveSymbolFunc != nil {
		return m.RemoveSymbolFunc(ctx, userID, symbolCode)
	}
	return nil
}

func (m *mockUsecase) ReorderSymbols(ctx context.Context, userID int64, orderedCodes []string) error {
	if m.ReorderSymbolsFunc != nil {
		return m.ReorderSymbolsFunc(ctx, userID, orderedCodes)
	}
	return nil
}

// newRouter は ContextUserID を注入するミドルウェア付きの gin ルーターを構築します。
func newRouter(t *testing.T, register func(r *gin.Engine)) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(jwtmw.ContextUserID, testUserID)
	})
	register(r)
	return r
}

func TestWatchlistHandler_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockList       func(ctx context.Context, userID int64) ([]watchlist.UserSymbol, error)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success: returns watchlist items",
			mockList: func(ctx context.Context, userID int64) ([]watchlist.UserSymbol, error) {
				assert.Equal(t, testUserID, userID)
				return []watchlist.UserSymbol{
					{ID: 1, UserID: testUserID, SymbolCode: "AAPL", SortKey: 0},
					{ID: 2, UserID: testUserID, SymbolCode: "MSFT", SortKey: 1},
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":1,"symbol_code":"AAPL","sort_key":0},{"id":2,"symbol_code":"MSFT","sort_key":1}]`,
		},
		{
			name: "success: empty watchlist returns empty array",
			mockList: func(ctx context.Context, userID int64) ([]watchlist.UserSymbol, error) {
				return []watchlist.UserSymbol{}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[]`,
		},
		{
			name: "error: usecase returns error",
			mockList: func(ctx context.Context, userID int64) ([]watchlist.UserSymbol, error) {
				return nil, errors.New("db failure")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal server error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUC := &mockUsecase{ListUserSymbolsFunc: tt.mockList}
			h := watchlisthttp.NewHandler(mockUC)
			router := newRouter(t, func(r *gin.Engine) {
				r.GET("/watchlist", h.List)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/watchlist", io.NopCloser(bytes.NewReader(nil)))
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestWatchlistHandler_Add(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		body           string
		mockAdd        func(ctx context.Context, userID int64, symbolCode string) error
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success: symbol added",
			body: `{"symbol_code":"AAPL"}`,
			mockAdd: func(ctx context.Context, userID int64, symbolCode string) error {
				assert.Equal(t, testUserID, userID)
				assert.Equal(t, "AAPL", symbolCode)
				return nil
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"message":"added to watchlist"}`,
		},
		{
			name: "error: symbol not found",
			body: `{"symbol_code":"XXXX"}`,
			mockAdd: func(ctx context.Context, userID int64, symbolCode string) error {
				return watchlist.ErrSymbolNotFound
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"symbol not found"}`,
		},
		{
			name: "error: already in watchlist",
			body: `{"symbol_code":"AAPL"}`,
			mockAdd: func(ctx context.Context, userID int64, symbolCode string) error {
				return watchlist.ErrAlreadyInWatchlist
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   `{"error":"symbol already in watchlist"}`,
		},
		{
			name:           "error: invalid request body returns 400",
			body:           `{"symbol_code":""}`,
			mockAdd:        nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "",
		},
		{
			name: "error: usecase returns internal error",
			body: `{"symbol_code":"AAPL"}`,
			mockAdd: func(ctx context.Context, userID int64, symbolCode string) error {
				return errors.New("db failure")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal server error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUC := &mockUsecase{AddSymbolFunc: tt.mockAdd}
			h := watchlisthttp.NewHandler(mockUC)
			router := newRouter(t, func(r *gin.Engine) {
				r.POST("/watchlist", h.Add)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/watchlist", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestWatchlistHandler_Remove(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		code           string
		mockRemove     func(ctx context.Context, userID int64, symbolCode string) error
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success: symbol removed",
			code: "AAPL",
			mockRemove: func(ctx context.Context, userID int64, symbolCode string) error {
				assert.Equal(t, testUserID, userID)
				assert.Equal(t, "AAPL", symbolCode)
				return nil
			},
			expectedStatus: http.StatusNoContent,
			expectedBody:   "",
		},
		{
			name: "error: not in watchlist",
			code: "AAPL",
			mockRemove: func(ctx context.Context, userID int64, symbolCode string) error {
				return watchlist.ErrNotInWatchlist
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"symbol not in watchlist"}`,
		},
		{
			name: "error: usecase returns internal error",
			code: "AAPL",
			mockRemove: func(ctx context.Context, userID int64, symbolCode string) error {
				return errors.New("db failure")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal server error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUC := &mockUsecase{RemoveSymbolFunc: tt.mockRemove}
			h := watchlisthttp.NewHandler(mockUC)
			router := newRouter(t, func(r *gin.Engine) {
				r.DELETE("/watchlist/:code", h.Remove)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodDelete, "/watchlist/"+tt.code, io.NopCloser(bytes.NewReader(nil)))
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestWatchlistHandler_Reorder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		body           string
		mockReorder    func(ctx context.Context, userID int64, orderedCodes []string) error
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success: watchlist reordered",
			body: `{"codes":["MSFT","AAPL"]}`,
			mockReorder: func(ctx context.Context, userID int64, orderedCodes []string) error {
				assert.Equal(t, testUserID, userID)
				assert.Equal(t, []string{"MSFT", "AAPL"}, orderedCodes)
				return nil
			},
			expectedStatus: http.StatusNoContent,
			expectedBody:   "",
		},
		{
			name:           "error: invalid request body returns 400",
			body:           `{"codes":[]}`,
			mockReorder:    nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "",
		},
		{
			name: "error: usecase returns internal error",
			body: `{"codes":["AAPL"]}`,
			mockReorder: func(ctx context.Context, userID int64, orderedCodes []string) error {
				return errors.New("db failure")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal server error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUC := &mockUsecase{ReorderSymbolsFunc: tt.mockReorder}
			h := watchlisthttp.NewHandler(mockUC)
			router := newRouter(t, func(r *gin.Engine) {
				r.PUT("/watchlist/reorder", h.Reorder)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPut, "/watchlist/reorder", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}
