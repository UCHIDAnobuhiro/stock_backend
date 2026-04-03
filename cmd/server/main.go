package main

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"stock_backend/internal/app/router"
	authadapters "stock_backend/internal/feature/auth/adapters"
	authentity "stock_backend/internal/feature/auth/domain/entity"
	authhandler "stock_backend/internal/feature/auth/transport/handler"
	authusecase "stock_backend/internal/feature/auth/usecase"
	candlesadapters "stock_backend/internal/feature/candles/adapters"
	candleshandler "stock_backend/internal/feature/candles/transport/handler"
	candlesusecase "stock_backend/internal/feature/candles/usecase"
	logogemini "stock_backend/internal/feature/logodetection/adapters/gemini"
	logovision "stock_backend/internal/feature/logodetection/adapters/vision"
	logohandler "stock_backend/internal/feature/logodetection/transport/handler"
	logousecase "stock_backend/internal/feature/logodetection/usecase"
	symbollistadapters "stock_backend/internal/feature/symbollist/adapters"
	symbolentity "stock_backend/internal/feature/symbollist/domain/entity"
	symbollisthandler "stock_backend/internal/feature/symbollist/transport/handler"
	symbollistusecase "stock_backend/internal/feature/symbollist/usecase"
	watchlistadapters "stock_backend/internal/feature/watchlist/adapters"
	watchlistentity "stock_backend/internal/feature/watchlist/domain/entity"
	watchlisthandler "stock_backend/internal/feature/watchlist/transport/handler"
	watchlistusecase "stock_backend/internal/feature/watchlist/usecase"
	"stock_backend/internal/platform/cache"
	infradb "stock_backend/internal/platform/db"
	jwtmw "stock_backend/internal/platform/jwt"
	"stock_backend/internal/platform/ratelimit"
	infraredis "stock_backend/internal/platform/redis"
)

func main() {
	// 構造化ロガーを初期化
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "DEBUG" {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// データベース接続
	db := infradb.OpenDB()

	// マイグレーション
	if os.Getenv("RUN_MIGRATIONS") == "true" {
		if err := infradb.RunMigrations(db,
			&authentity.User{},
			&candlesadapters.CandleModel{},
			&symbolentity.Symbol{},
			&watchlistentity.UserSymbol{},
		); err != nil {
			slog.Error("failed to migrate", "error", err)
			os.Exit(1)
		}
		if err := addWatchlistFKConstraints(db); err != nil {
			slog.Error("failed to add watchlist FK constraints", "error", err)
			os.Exit(1)
		}
	}

	// Redis接続
	var rdb *redisv9.Client
	if tmp, err := infraredis.NewRedisClient(); err != nil {
		slog.Warn("Redis unavailable, running without cache", "error", err)
		rdb = nil
	} else {
		rdb = tmp
		defer func() {
			if err := rdb.Close(); err != nil {
				slog.Error("Failed to close Redis client", "error", err)
			}
		}()
	}

	// リポジトリ
	userRepo := authadapters.NewUserRepository(db)
	symbolRepo := symbollistadapters.NewSymbolRepository(db)
	candleRepo := candlesadapters.NewCandleRepository(db)
	watchlistRepo := watchlistadapters.NewWatchlistRepository(db)

	// Redisキャッシュでラップ
	ttl := cache.TimeUntilNext8AM()
	cachedCandleRepo := candlesadapters.NewCachingCandleRepository(rdb, ttl, candleRepo, "candles")

	// JWTジェネレータ
	jwtSecret := os.Getenv(jwtmw.EnvKeyJWTSecret)
	if jwtSecret == "" {
		slog.Error("JWT_SECRET environment variable is required")
		os.Exit(1)
	}
	jwtGen := jwtmw.NewGenerator(jwtSecret, 1*time.Hour)

	// パスワードペッパー
	passwordPepper := os.Getenv(authusecase.EnvKeyPasswordPepper)
	if passwordPepper == "" {
		slog.Error("PASSWORD_PEPPER environment variable is required")
		os.Exit(1)
	}

	// Google Cloudクライアント初期化
	visionDetector, err := logovision.NewVisionLogoDetector(context.Background())
	if err != nil {
		slog.Error("failed to create vision client", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := visionDetector.Close(); err != nil {
			slog.Error("Failed to close vision client", "error", err)
		}
	}()

	geminiAnalyzer, err := logogemini.NewGeminiAnalyzer(context.Background())
	if err != nil {
		slog.Error("failed to create gemini client", "error", err)
		os.Exit(1)
	}

	// レートリミッター
	rateLimiter := ratelimit.NewLimiter(rdb)

	// ユースケース
	authUC := authusecase.NewAuthUsecase(userRepo, jwtGen, passwordPepper)
	symbolUC := symbollistusecase.NewSymbolUsecase(symbolRepo)
	candlesUC := candlesusecase.NewCandlesUsecase(cachedCandleRepo)
	logoUC := logousecase.NewLogoDetectionUsecase(visionDetector, geminiAnalyzer)
	watchlistUC := watchlistusecase.NewWatchlistUsecase(watchlistRepo, symbolRepo)

	// ハンドラー
	authH := authhandler.NewAuthHandler(authUC, rateLimiter, watchlistUC)
	symbolH := symbollisthandler.NewSymbolHandler(symbolUC)
	candlesH := candleshandler.NewCandlesHandler(candlesUC)
	logoH := logohandler.NewLogoDetectionHandler(logoUC)
	watchlistH := watchlisthandler.NewWatchlistHandler(watchlistUC)

	// CORS許可オリジンを環境変数から読み込む（デフォルト: http://localhost:3000）
	corsOrigins := []string{"http://localhost:3000"}
	if raw := os.Getenv("CORS_ALLOWED_ORIGINS"); raw != "" {
		parts := strings.Split(raw, ",")
		corsOrigins = make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				corsOrigins = append(corsOrigins, trimmed)
			}
		}
	}

	// ルーター作成
	r := router.NewRouter(authH, candlesH, symbolH, logoH, watchlistH, rateLimiter, corsOrigins)

	slog.Info("Starting server", "port", 8080)
	if err := r.Run(":8080"); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}

// addWatchlistFKConstraints はwatchlistsテーブルのFK制約を冪等に追加します。
// GORMのAutoMigrateはFK制約を自動生成しないため、マイグレーション後に明示的に実行します。
func addWatchlistFKConstraints(db *gorm.DB) error {
	if !db.Migrator().HasConstraint(&watchlistentity.UserSymbol{}, "fk_watchlists_user") {
		if err := db.Exec(`ALTER TABLE watchlists ADD CONSTRAINT fk_watchlists_user
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`).Error; err != nil {
			return err
		}
		slog.Info("added FK constraint: fk_watchlists_user")
	}
	if !db.Migrator().HasConstraint(&watchlistentity.UserSymbol{}, "fk_watchlists_symbol") {
		if err := db.Exec(`ALTER TABLE watchlists ADD CONSTRAINT fk_watchlists_symbol
			FOREIGN KEY (symbol_code) REFERENCES symbols(code) ON DELETE RESTRICT`).Error; err != nil {
			return err
		}
		slog.Info("added FK constraint: fk_watchlists_symbol")
	}
	return nil
}
