package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func setupRouter() *gin.Engine {
	r := gin.New()
	r.GET("/healthz", Health)
	r.HEAD("/healthz", Health)
	r.OPTIONS("/healthz", Health)
	r.POST("/healthz", Health)
	r.PUT("/healthz", Health)
	r.DELETE("/healthz", Health)
	return r
}

func TestHealth_GET(t *testing.T) {
	t.Parallel()

	router := setupRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check response body
	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if response["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", response["status"])
	}

	// Check Cache-Control header
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("expected Cache-Control 'no-store', got %q", w.Header().Get("Cache-Control"))
	}
}

func TestHealth_HEAD(t *testing.T) {
	t.Parallel()

	router := setupRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodHead, "/healthz", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// HEAD should have no body
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body for HEAD request, got %d bytes", w.Body.Len())
	}

	// Check Cache-Control header
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("expected Cache-Control 'no-store', got %q", w.Header().Get("Cache-Control"))
	}
}

func TestHealth_OPTIONS(t *testing.T) {
	t.Parallel()

	router := setupRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/healthz", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Check Cache-Control header
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("expected Cache-Control 'no-store', got %q", w.Header().Get("Cache-Control"))
	}
}

func TestHealth_POST(t *testing.T) {
	t.Parallel()

	router := setupRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)

	router.ServeHTTP(w, req)

	// Default case returns JSON
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if response["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", response["status"])
	}
}

func TestHealth_AllMethods_CacheControl(t *testing.T) {
	t.Parallel()

	methods := []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
	}

	router := setupRouter()

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			req := httptest.NewRequest(method, "/healthz", nil)

			router.ServeHTTP(w, req)

			// All methods should have Cache-Control header
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Errorf("expected Cache-Control 'no-store', got %q", w.Header().Get("Cache-Control"))
			}
		})
	}
}

func TestHealth_ResponseStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method         string
		expectedStatus int
	}{
		{http.MethodGet, http.StatusOK},
		{http.MethodHead, http.StatusOK},
		{http.MethodOptions, http.StatusNoContent},
		{http.MethodPost, http.StatusOK},
		{http.MethodPut, http.StatusOK},
		{http.MethodDelete, http.StatusOK},
	}

	router := setupRouter()

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/healthz", nil)

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
