package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	redisv9 "github.com/redis/go-redis/v9"

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

// signupWithDefaults はサインアップ時にデフォルトウォッチリストを初期化するラッパーです。
// authhandler.AuthUsecaseインターフェースを満たします。
type signupWithDefaults struct {
	auth      authhandler.AuthUsecase
	watchlist *watchlistusecase.WatchlistUsecase
}

func (s *signupWithDefaults) Signup(ctx context.Context, email, password string) (uint, error) {
	userID, err := s.auth.Signup(ctx, email, password)
	if err != nil {
		return 0, err
	}
	if initErr := s.watchlist.InitializeDefaults(ctx, userID); initErr != nil {
		slog.Error("failed to initialize default watchlist, rolling back user creation",
			"userID", userID, "error", initErr)
		if delErr := s.auth.DeleteUser(ctx, userID); delErr != nil {
			slog.Error("failed to delete user during signup rollback", "userID", userID, "error", delErr)
		}
		return 0, fmt.Errorf("failed to initialize watchlist: %w", initErr)
	}
	return userID, nil
}

func (s *signupWithDefaults) Login(ctx context.Context, email, password string) (string, error) {
	return s.auth.Login(ctx, email, password)
}

func (s *signupWithDefaults) DeleteUser(ctx context.Context, id uint) error {
	return s.auth.DeleteUser(ctx, id)
}

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
	userRepo := authadapters.NewUserMySQL(db)
	symbolRepo := symbollistadapters.NewSymbolRepository(db)
	candleRepo := candlesadapters.NewCandleRepository(db)
	userSymbolRepo := watchlistadapters.NewUserSymbolRepository(db)

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
	watchlistUC := watchlistusecase.NewWatchlistUsecase(userSymbolRepo)

	// サインアップ時にデフォルトウォッチリストを初期化するラッパー
	authWithDefaults := &signupWithDefaults{auth: authUC, watchlist: watchlistUC}

	// ハンドラー
	authH := authhandler.NewAuthHandler(authWithDefaults, rateLimiter)
	symbolH := symbollisthandler.NewSymbolHandler(symbolUC)
	candlesH := candleshandler.NewCandlesHandler(candlesUC)
	logoH := logohandler.NewLogoDetectionHandler(logoUC)
	watchlistH := watchlisthandler.NewWatchlistHandler(watchlistUC)

	// ルーター作成
	router := router.NewRouter(authH, candlesH, symbolH, logoH, watchlistH, rateLimiter)

	// モバイルアプリ向けのためCORSミドルウェアはコメントアウト
	// router.Use(cors.Default())

	slog.Info("Starting server", "port", 8080)
	if err := router.Run(":8080"); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
