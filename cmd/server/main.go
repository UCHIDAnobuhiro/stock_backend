package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	redisv9 "github.com/redis/go-redis/v9"

	"stock_backend/internal/app/config"
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
			&authentity.OAuthAccount{},
			&candlesadapters.CandleModel{},
			&symbolentity.Symbol{},
			&watchlistentity.UserSymbol{},
		); err != nil {
			slog.Error("failed to migrate", "error", err)
			os.Exit(1)
		}
		if err := authadapters.MakePasswordNullable(db); err != nil {
			slog.Error("failed to make password nullable", "error", err)
			os.Exit(1)
		}
		if err := authadapters.AddOAuthAccountsFKConstraints(db); err != nil {
			slog.Error("failed to add oauth_accounts FK constraints", "error", err)
			os.Exit(1)
		}
		if err := watchlistadapters.AddFKConstraints(db); err != nil {
			slog.Error("failed to add watchlist FK constraints", "error", err)
			os.Exit(1)
		}
		if err := candlesadapters.AddFKConstraints(db); err != nil {
			slog.Error("failed to add candles FK constraints", "error", err)
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

	// Redisキャッシュでラップ（TTLはingest連続失敗時のセーフティネット、通常は日次ingestで上書き）
	cachedCandleRepo := candlesadapters.NewCachingCandleRepository(rdb, candlesadapters.DefaultCacheTTL, candleRepo, "candles")

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

	// COOKIE_SECURE を優先し、未設定なら APP_ENV=production をフォールバックとして使用
	cookieSecureRaw := os.Getenv("COOKIE_SECURE")
	defaultSecure := os.Getenv("APP_ENV") == "production"
	secureCookie, ok := config.ParseBoolString(cookieSecureRaw, defaultSecure)
	if !ok {
		slog.Warn("invalid COOKIE_SECURE value, falling back to default", "value", cookieSecureRaw, "default", secureCookie)
	}

	// OAuth ハンドラー（環境変数が未設定の場合はOAuth機能なしで起動）
	var oauthH *authhandler.OAuthHandler
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	oauthEnabled := googleClientID != "" || githubClientID != ""
	if oauthEnabled {
		if rdb == nil {
			slog.Error("OAuth requires Redis but Redis is unavailable")
			os.Exit(1)
		}
		oauthFrontendURL := os.Getenv("OAUTH_FRONTEND_REDIRECT_URL")
		if oauthFrontendURL == "" {
			slog.Error("OAUTH_FRONTEND_REDIRECT_URL is required when OAuth is enabled")
			os.Exit(1)
		}
		oauthProviders := map[string]authusecase.OAuthProvider{}
		if googleClientID != "" {
			googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
			if googleClientSecret == "" {
				slog.Error("GOOGLE_CLIENT_SECRET is required when GOOGLE_CLIENT_ID is set")
				os.Exit(1)
			}
			googleRedirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
			if googleRedirectURL == "" {
				slog.Error("GOOGLE_REDIRECT_URL is required when GOOGLE_CLIENT_ID is set")
				os.Exit(1)
			}
			oauthProviders["google"] = authadapters.NewGoogleProvider(
				googleClientID,
				googleClientSecret,
				googleRedirectURL,
				&http.Client{Timeout: 10 * time.Second},
			)
		}
		if githubClientID != "" {
			githubClientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
			if githubClientSecret == "" {
				slog.Error("GITHUB_CLIENT_SECRET is required when GITHUB_CLIENT_ID is set")
				os.Exit(1)
			}
			githubRedirectURL := os.Getenv("GITHUB_REDIRECT_URL")
			if githubRedirectURL == "" {
				slog.Error("GITHUB_REDIRECT_URL is required when GITHUB_CLIENT_ID is set")
				os.Exit(1)
			}
			oauthProviders["github"] = authadapters.NewGitHubProvider(
				githubClientID,
				githubClientSecret,
				githubRedirectURL,
				&http.Client{Timeout: 10 * time.Second},
			)
		}
		oauthUC := authusecase.NewOAuthUsecase(
			userRepo,
			authadapters.NewOAuthAccountRepository(db),
			userRepo,
			authadapters.NewRedisOAuthStateStore(rdb),
			jwtGen,
			oauthProviders,
			watchlistUC,
		)
		oauthH = authhandler.NewOAuthHandler(oauthUC, secureCookie, oauthFrontendURL)
	}

	// ハンドラー
	authH := authhandler.NewAuthHandler(authUC, rateLimiter, secureCookie, watchlistUC)
	symbolH := symbollisthandler.NewSymbolHandler(symbolUC)
	candlesH := candleshandler.NewCandlesHandler(candlesUC)
	logoH := logohandler.NewLogoDetectionHandler(logoUC)
	watchlistH := watchlisthandler.NewWatchlistHandler(watchlistUC)

	// CORS許可オリジンを環境変数から読み込む（デフォルト: http://localhost:3000）
	corsOrigins := config.ParseCORSOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if corsOrigins == nil {
		corsOrigins = []string{"http://localhost:3000"}
	}

	// ルーター作成
	r := router.NewRouter(authH, oauthH, candlesH, symbolH, logoH, watchlistH, rateLimiter, corsOrigins)

	slog.Info("Starting server", "port", 8080)
	if err := r.Run(":8080"); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
