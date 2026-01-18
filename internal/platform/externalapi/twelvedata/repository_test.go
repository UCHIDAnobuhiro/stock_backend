package twelvedata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewTwelveDataMarket(t *testing.T) {
	t.Parallel()

	cfg := Config{
		TwelveDataAPIKey: "test-key",
		BaseURL:          "https://api.test.com",
		Timeout:          10 * time.Second,
	}
	client := &http.Client{}

	market := NewTwelveDataMarket(cfg, client)

	if market == nil {
		t.Fatal("expected non-nil market")
	}
	if market.cfg.TwelveDataAPIKey != cfg.TwelveDataAPIKey {
		t.Errorf("expected API key %q, got %q", cfg.TwelveDataAPIKey, market.cfg.TwelveDataAPIKey)
	}
}

func TestTwelveDataMarket_GetTimeSeries_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters
		if r.URL.Query().Get("symbol") != "AAPL" {
			t.Errorf("expected symbol AAPL, got %s", r.URL.Query().Get("symbol"))
		}
		if r.URL.Query().Get("interval") != "1day" {
			t.Errorf("expected interval 1day, got %s", r.URL.Query().Get("interval"))
		}
		if r.URL.Query().Get("outputsize") != "100" {
			t.Errorf("expected outputsize 100, got %s", r.URL.Query().Get("outputsize"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"status": "ok",
			"symbol": "AAPL",
			"interval": "1day",
			"values": [
				{
					"datetime": "2025-01-15",
					"open": "150.00",
					"high": "155.00",
					"low": "149.00",
					"close": "154.50",
					"volume": "1000000"
				},
				{
					"datetime": "2025-01-14 09:30:00",
					"open": "148.00",
					"high": "151.00",
					"low": "147.50",
					"close": "150.00",
					"volume": "900000"
				}
			]
		}`))
	}))
	defer server.Close()

	cfg := Config{
		TwelveDataAPIKey: "test-key",
		BaseURL:          server.URL,
	}
	market := NewTwelveDataMarket(cfg, server.Client())

	candles, err := market.GetTimeSeries(context.Background(), "AAPL", "1day", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(candles) != 2 {
		t.Fatalf("expected 2 candles, got %d", len(candles))
	}

	// Check first candle
	if candles[0].Open != 150.00 {
		t.Errorf("expected open 150.00, got %f", candles[0].Open)
	}
	if candles[0].Close != 154.50 {
		t.Errorf("expected close 154.50, got %f", candles[0].Close)
	}
	if candles[0].Volume != 1000000 {
		t.Errorf("expected volume 1000000, got %d", candles[0].Volume)
	}
}

func TestTwelveDataMarket_GetTimeSeries_HTTPError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
	}{
		{"bad request", http.StatusBadRequest},
		{"unauthorized", http.StatusUnauthorized},
		{"forbidden", http.StatusForbidden},
		{"not found", http.StatusNotFound},
		{"internal server error", http.StatusInternalServerError},
		{"service unavailable", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			cfg := Config{
				TwelveDataAPIKey: "test-key",
				BaseURL:          server.URL,
			}
			market := NewTwelveDataMarket(cfg, server.Client())

			_, err := market.GetTimeSeries(context.Background(), "AAPL", "1day", 100)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "twelvedata http") {
				t.Errorf("expected HTTP error message, got %v", err)
			}
		})
	}
}

func TestTwelveDataMarket_GetTimeSeries_APIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"status": "error",
			"message": "Invalid API key"
		}`))
	}))
	defer server.Close()

	cfg := Config{
		TwelveDataAPIKey: "invalid-key",
		BaseURL:          server.URL,
	}
	market := NewTwelveDataMarket(cfg, server.Client())

	_, err := market.GetTimeSeries(context.Background(), "AAPL", "1day", 100)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Invalid API key") {
		t.Errorf("expected API error message, got %v", err)
	}
}

func TestTwelveDataMarket_GetTimeSeries_InvalidJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	cfg := Config{
		TwelveDataAPIKey: "test-key",
		BaseURL:          server.URL,
	}
	market := NewTwelveDataMarket(cfg, server.Client())

	_, err := market.GetTimeSeries(context.Background(), "AAPL", "1day", 100)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTwelveDataMarket_GetTimeSeries_InvalidDateTime(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"status": "ok",
			"values": [
				{
					"datetime": "invalid-date",
					"open": "150.00",
					"high": "155.00",
					"low": "149.00",
					"close": "154.50",
					"volume": "1000000"
				}
			]
		}`))
	}))
	defer server.Close()

	cfg := Config{
		TwelveDataAPIKey: "test-key",
		BaseURL:          server.URL,
	}
	market := NewTwelveDataMarket(cfg, server.Client())

	_, err := market.GetTimeSeries(context.Background(), "AAPL", "1day", 100)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "parse time") {
		t.Errorf("expected parse time error, got %v", err)
	}
}

func TestTwelveDataMarket_GetTimeSeries_InvalidNumbers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response string
		errField string
	}{
		{
			name: "invalid open",
			response: `{
				"status": "ok",
				"values": [{"datetime": "2025-01-15", "open": "abc", "high": "155.00", "low": "149.00", "close": "154.50", "volume": "1000000"}]
			}`,
			errField: "parse open",
		},
		{
			name: "invalid high",
			response: `{
				"status": "ok",
				"values": [{"datetime": "2025-01-15", "open": "150.00", "high": "xyz", "low": "149.00", "close": "154.50", "volume": "1000000"}]
			}`,
			errField: "parse high",
		},
		{
			name: "invalid low",
			response: `{
				"status": "ok",
				"values": [{"datetime": "2025-01-15", "open": "150.00", "high": "155.00", "low": "bad", "close": "154.50", "volume": "1000000"}]
			}`,
			errField: "parse low",
		},
		{
			name: "invalid close",
			response: `{
				"status": "ok",
				"values": [{"datetime": "2025-01-15", "open": "150.00", "high": "155.00", "low": "149.00", "close": "bad", "volume": "1000000"}]
			}`,
			errField: "parse close",
		},
		{
			name: "invalid volume",
			response: `{
				"status": "ok",
				"values": [{"datetime": "2025-01-15", "open": "150.00", "high": "155.00", "low": "149.00", "close": "154.50", "volume": "not-a-number"}]
			}`,
			errField: "parse volume",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			cfg := Config{
				TwelveDataAPIKey: "test-key",
				BaseURL:          server.URL,
			}
			market := NewTwelveDataMarket(cfg, server.Client())

			_, err := market.GetTimeSeries(context.Background(), "AAPL", "1day", 100)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errField) {
				t.Errorf("expected error containing %q, got %v", tt.errField, err)
			}
		})
	}
}

func TestTwelveDataMarket_GetTimeSeries_EmptyValues(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"status": "ok",
			"values": []
		}`))
	}))
	defer server.Close()

	cfg := Config{
		TwelveDataAPIKey: "test-key",
		BaseURL:          server.URL,
	}
	market := NewTwelveDataMarket(cfg, server.Client())

	candles, err := market.GetTimeSeries(context.Background(), "AAPL", "1day", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candles) != 0 {
		t.Errorf("expected 0 candles, got %d", len(candles))
	}
}

func TestTwelveDataMarket_GetTimeSeries_ContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{
		TwelveDataAPIKey: "test-key",
		BaseURL:          server.URL,
	}
	market := NewTwelveDataMarket(cfg, server.Client())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := market.GetTimeSeries(ctx, "AAPL", "1day", 100)
	if err == nil {
		t.Fatal("expected error due to context cancellation, got nil")
	}
}

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	// Note: This test doesn't set environment variables to avoid affecting other tests
	cfg := LoadConfig()

	if cfg.Timeout != 10*time.Second {
		t.Errorf("expected timeout 10s, got %v", cfg.Timeout)
	}
}
