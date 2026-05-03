package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"stock_backend/internal/app/di"
	symbollistadapters "stock_backend/internal/feature/symbollist/adapters"
	symbollistusecase "stock_backend/internal/feature/symbollist/usecase"
	"stock_backend/internal/platform/db"
	"stock_backend/internal/shared/ratelimiter"
)

const (
	logoRateLimitPerMinute = 7
	defaultMaxFailureRate  = 0.2
)

func shouldFailExit(result symbollistusecase.LogoIngestResult, threshold float64) bool {
	return result.FailureRate() > threshold
}

func main() {
	os.Exit(run())
}

func run() int {
	db := db.OpenDB()
	logoProvider := di.NewMarket()
	symbolRepo := symbollistadapters.NewSymbolRepository(db)
	rateLimiter := ratelimiter.NewRateLimiter(logoRateLimitPerMinute, time.Minute)
	uc := symbollistusecase.NewLogoIngestUsecase(logoProvider, symbolRepo, rateLimiter)

	timeoutHours := 3
	if v := os.Getenv("LOGO_INGEST_TIMEOUT_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutHours = n
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutHours)*time.Hour)
	defer cancel()

	maxFailureRate := defaultMaxFailureRate
	if v := os.Getenv("LOGO_INGEST_MAX_FAILURE_RATE"); v != "" {
		if r, err := strconv.ParseFloat(v, 64); err == nil && r >= 0 && r <= 1 {
			maxFailureRate = r
		} else {
			slog.Warn("invalid LOGO_INGEST_MAX_FAILURE_RATE, using default", "value", v, "default", defaultMaxFailureRate)
		}
	}

	start := time.Now()
	result, err := uc.IngestAll(ctx)
	duration := time.Since(start)

	slog.Info("logo ingest summary",
		"total", result.Total,
		"succeeded", result.Succeeded,
		"failed", result.Failed,
		"failure_rate", result.FailureRate(),
		"duration", duration.String(),
	)

	if err != nil {
		slog.Error("logo ingest aborted by fatal error", "error", err)
		return 1
	}
	if shouldFailExit(result, maxFailureRate) {
		slog.Error("logo ingest failure rate exceeded threshold",
			"failure_rate", result.FailureRate(),
			"threshold", maxFailureRate,
		)
		return 1
	}
	slog.Info("logo ingest ok")
	return 0
}
