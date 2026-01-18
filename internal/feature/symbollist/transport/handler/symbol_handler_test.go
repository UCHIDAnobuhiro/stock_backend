package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"stock_backend/internal/feature/symbollist/domain/entity"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// mockSymbolUsecase is a mock implementation of SymbolUsecase interface.
type mockSymbolUsecase struct {
	ListActiveSymbolsFunc func(ctx context.Context) ([]entity.Symbol, error)
}

func (m *mockSymbolUsecase) ListActiveSymbols(ctx context.Context) ([]entity.Symbol, error) {
	if m.ListActiveSymbolsFunc != nil {
		return m.ListActiveSymbolsFunc(ctx)
	}
	return nil, nil
}

func TestNewSymbolHandler(t *testing.T) {
	t.Parallel()

	mockUC := &mockSymbolUsecase{}
	handler := NewSymbolHandler(mockUC)

	assert.NotNil(t, handler, "handler should not be nil")
	assert.NotNil(t, handler.uc, "usecase should not be nil")
}

func TestSymbolHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name               string
		mockListActiveFunc func(ctx context.Context) ([]entity.Symbol, error)
		expectedStatus     int
		expectedBody       string
	}{
		{
			name: "success: returns list of symbols",
			mockListActiveFunc: func(ctx context.Context) ([]entity.Symbol, error) {
				return []entity.Symbol{
					{ID: 1, Code: "7203.T", Name: "Toyota Motor", Market: "TSE", IsActive: true, SortKey: 1},
					{ID: 2, Code: "6758.T", Name: "Sony Group", Market: "TSE", IsActive: true, SortKey: 2},
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"code":"7203.T","name":"Toyota Motor"},{"code":"6758.T","name":"Sony Group"}]`,
		},
		{
			name: "success: returns empty list when no symbols",
			mockListActiveFunc: func(ctx context.Context) ([]entity.Symbol, error) {
				return []entity.Symbol{}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[]`,
		},
		{
			name: "success: returns single symbol",
			mockListActiveFunc: func(ctx context.Context) ([]entity.Symbol, error) {
				return []entity.Symbol{
					{ID: 1, Code: "9984.T", Name: "SoftBank Group", Market: "TSE", IsActive: true, SortKey: 1},
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"code":"9984.T","name":"SoftBank Group"}]`,
		},
		{
			name: "failure: usecase returns error",
			mockListActiveFunc: func(ctx context.Context) ([]entity.Symbol, error) {
				return nil, errors.New("database connection failed")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"database connection failed"}`,
		},
		{
			name: "success: returns nil from usecase",
			mockListActiveFunc: func(ctx context.Context) ([]entity.Symbol, error) {
				return nil, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUC := &mockSymbolUsecase{
				ListActiveSymbolsFunc: tt.mockListActiveFunc,
			}
			handler := NewSymbolHandler(mockUC)

			router := gin.New()
			router.GET("/symbols", handler.List)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/symbols", nil)

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestSymbolHandler_List_DTOConversion(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	// Verify that only code and name are included in the response (not ID, Market, IsActive, SortKey)
	mockUC := &mockSymbolUsecase{
		ListActiveSymbolsFunc: func(ctx context.Context) ([]entity.Symbol, error) {
			return []entity.Symbol{
				{
					ID:       999,
					Code:     "TEST.T",
					Name:     "Test Company",
					Market:   "NYSE",
					IsActive: true,
					SortKey:  100,
				},
			}, nil
		},
	}
	handler := NewSymbolHandler(mockUC)

	router := gin.New()
	router.GET("/symbols", handler.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/symbols", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Response should only contain code and name fields
	assert.JSONEq(t, `[{"code":"TEST.T","name":"Test Company"}]`, w.Body.String())
	// Verify that internal fields are not exposed
	assert.NotContains(t, w.Body.String(), "999")
	assert.NotContains(t, w.Body.String(), "NYSE")
	assert.NotContains(t, w.Body.String(), "is_active")
	assert.NotContains(t, w.Body.String(), "sort_key")
}
