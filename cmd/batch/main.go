package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	redisv9 "github.com/redis/go-redis/v9"

	"stock_backend/internal/app/di"
	candlesadapters "stock_backend/internal/feature/candles/adapters"
	candlesusecase "stock_backend/internal/feature/candles/usecase"
	symbollistadapters "stock_backend/internal/feature/symbollist/adapters"
	symbollistusecase "stock_backend/internal/feature/symbollist/usecase"
	"stock_backend/internal/platform/db"
	infraredis "stock_backend/internal/platform/redis"
	"stock_backend/internal/shared/clientratelimit"
)

const (
	rateLimitPerMinute    = 7   // TwelveData APIのレートリミット（無料枠上限8/分、固定ウィンドウずれ対策で1つ余裕を持たせる）
	defaultMaxFailureRate = 0.2 // *_MAX_FAILURE_RATE のデフォルト値
)

// failureRater は ingest 系 result が共通で実装する失敗率取得インターフェース。
type failureRater interface {
	FailureRate() float64
}

// shouldFailExit は失敗率しきい値から非ゼロ終了すべきかを判定する。
// しきい値ちょうど（FailureRate == threshold）は許容し、超過時のみ true を返す。
func shouldFailExit(result failureRater, threshold float64) bool {
	return result.FailureRate() > threshold
}

// parseTimeoutHours は env のタイムアウト時間（正の整数）を読み取る。未設定・不正時は def を返す。
func parseTimeoutHours(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// parseMaxFailureRate は env の失敗率しきい値（[0,1]）を読み取る。不正時は警告して def を返す。
func parseMaxFailureRate(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if r, err := strconv.ParseFloat(v, 64); err == nil && r >= 0 && r <= 1 {
			return r
		}
		slog.Warn("invalid max failure rate, using default", "key", key, "value", v, "default", def)
	}
	return def
}

// main は run の戻り値で os.Exit するだけのラッパー。
// os.Exit は defer を実行しないため、Redis Close 等の後処理が走るよう
// 実体は run に分離している。
func main() {
	os.Exit(run(os.Args[1:]))
}

// run は job_id（コマンド引数）に応じてバッチを実行し、終了コードを返す。
// candles: 株価取り込み、logo: ロゴURL取り込み。
func run(args []string) int {
	if len(args) < 1 {
		slog.Error("job_id is required", "usage", "batch <candles|logo>")
		return 2
	}
	switch args[0] {
	case "candles":
		return runCandleIngest()
	case "logo":
		return runLogoIngest()
	default:
		slog.Error("unknown job_id", "job_id", args[0], "supported", "candles, logo")
		return 2
	}
}

// runCandleIngest は TwelveData から株価データを取り込み、終了コード（0 or 1）を返す。
func runCandleIngest() int {
	sqlDB, err := db.OpenSQL()
	if err != nil {
		slog.Error("DB open failed", "error", err)
		return 1
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			slog.Warn("failed to close sqlDB", "error", err)
		}
	}()
	marketRepo := di.NewMarket()
	candleRepo := candlesadapters.NewCandleRepository(sqlDB)
	symbolRepo := symbollistadapters.NewSymbolRepository(sqlDB)
	ingestSymbolRepo := di.NewIngestSymbolAdapter(symbolRepo)
	rateLimiter := clientratelimit.NewRateLimiter(rateLimitPerMinute, time.Minute)

	// Redis接続（ベストエフォート: 接続失敗時はキャッシュウォームアップなしで続行）
	var rdb *redisv9.Client
	if tmp, err := infraredis.NewRedisClient(); err != nil {
		slog.Warn("Redis unavailable, cache warm-up disabled", "error", err)
	} else {
		rdb = tmp
		defer func() {
			if err := rdb.Close(); err != nil {
				slog.Error("Failed to close Redis client", "error", err)
			}
		}()
	}

	// TTLはingest連続失敗時のセーフティネット、通常は UpsertBatch で日次上書き
	cachedCandleRepo := candlesadapters.NewCachingCandleRepository(rdb, candlesadapters.DefaultCacheTTL, candleRepo, "candles")

	uc := candlesusecase.NewIngestUsecase(marketRepo, cachedCandleRepo, ingestSymbolRepo, rateLimiter)

	timeoutHours := parseTimeoutHours("INGEST_TIMEOUT_HOURS", 3)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutHours)*time.Hour)
	defer cancel()

	maxFailureRate := parseMaxFailureRate("INGEST_MAX_FAILURE_RATE", defaultMaxFailureRate)

	start := time.Now()
	result, err := uc.IngestAll(ctx)
	duration := time.Since(start)

	slog.Info("ingest summary",
		"total", result.Total,
		"succeeded", result.Succeeded,
		"failed", result.Failed,
		"failure_rate", result.FailureRate(),
		"duration", duration.String(),
	)

	if err != nil {
		slog.Error("ingest aborted by fatal error", "error", err)
		return 1
	}
	if shouldFailExit(result, maxFailureRate) {
		slog.Error("ingest failure rate exceeded threshold",
			"failure_rate", result.FailureRate(),
			"threshold", maxFailureRate,
		)
		return 1
	}
	slog.Info("ingest ok")
	return 0
}

// runLogoIngest は TwelveData からロゴURLを取り込み、終了コード（0 or 1）を返す。
func runLogoIngest() int {
	sqlDB, err := db.OpenSQL()
	if err != nil {
		slog.Error("DB open failed", "error", err)
		return 1
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			slog.Warn("failed to close sqlDB", "error", err)
		}
	}()
	logoProvider := di.NewMarket()
	symbolRepo := symbollistadapters.NewSymbolRepository(sqlDB)
	rateLimiter := clientratelimit.NewRateLimiter(rateLimitPerMinute, time.Minute)
	uc := symbollistusecase.NewLogoIngestUsecase(logoProvider, symbolRepo, rateLimiter)

	timeoutHours := parseTimeoutHours("LOGO_INGEST_TIMEOUT_HOURS", 3)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutHours)*time.Hour)
	defer cancel()

	maxFailureRate := parseMaxFailureRate("LOGO_INGEST_MAX_FAILURE_RATE", defaultMaxFailureRate)

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
