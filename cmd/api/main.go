package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	redisv9 "github.com/redis/go-redis/v9"

	"stock_backend/internal/app/config"
	"stock_backend/internal/app/router"
	authadapters "stock_backend/internal/feature/auth/adapters"
	authhandler "stock_backend/internal/feature/auth/transport/handler"
	authusecase "stock_backend/internal/feature/auth/usecase"
	"stock_backend/internal/feature/candles"
	candleshttp "stock_backend/internal/feature/candles/transport"
	logogemini "stock_backend/internal/feature/logodetection/adapters/gemini"
	logovision "stock_backend/internal/feature/logodetection/adapters/vision"
	logohandler "stock_backend/internal/feature/logodetection/transport/handler"
	logousecase "stock_backend/internal/feature/logodetection/usecase"
	symbollistadapters "stock_backend/internal/feature/symbollist/adapters"
	symbollisthandler "stock_backend/internal/feature/symbollist/transport/handler"
	symbollistusecase "stock_backend/internal/feature/symbollist/usecase"
	"stock_backend/internal/feature/watchlist"
	watchlisthttp "stock_backend/internal/feature/watchlist/transport"
	infradb "stock_backend/internal/platform/db"
	"stock_backend/internal/platform/httpratelimit"
	jwtmw "stock_backend/internal/platform/jwt"
	infraredis "stock_backend/internal/platform/redis"
)

// oauthProviderConfig は OAuth プロバイダ1社分の検証済み認証情報。
type oauthProviderConfig struct {
	clientID     string
	clientSecret string
	redirectURL  string
}

// oauthConfig は OAuth 機能の検証済み設定。いずれかのプロバイダが設定された場合のみ生成される。
type oauthConfig struct {
	frontendURL string
	google      *oauthProviderConfig // 未設定なら nil
	github      *oauthProviderConfig // 未設定なら nil
}

// serverConfig は環境変数から読み込んだ検証済みのサーバー設定。
type serverConfig struct {
	jwtSecret      string
	passwordPepper string
	secureCookie   bool
	corsOrigins    []string
	oauth          *oauthConfig // OAuth 無効なら nil
}

// loadServerConfig は環境変数を読み込んで検証し、検証済みの設定を返す。
// 外部接続（DB/Redis/GCP）に依存しない純粋関数なのでユニットテスト可能。
// 必須項目の欠落や OAuth 設定の不整合があればエラーを返す。
func loadServerConfig() (serverConfig, error) {
	jwtSecret := os.Getenv(jwtmw.EnvKeyJWTSecret)
	if jwtSecret == "" {
		return serverConfig{}, fmt.Errorf("%s is required", jwtmw.EnvKeyJWTSecret)
	}

	passwordPepper := os.Getenv(authusecase.EnvKeyPasswordPepper)
	if passwordPepper == "" {
		return serverConfig{}, fmt.Errorf("%s is required", authusecase.EnvKeyPasswordPepper)
	}

	// COOKIE_SECURE を優先し、未設定なら APP_ENV=production をフォールバックとして使用
	cookieSecureRaw := os.Getenv("COOKIE_SECURE")
	defaultSecure := os.Getenv("APP_ENV") == "production"
	secureCookie, ok := config.ParseBoolString(cookieSecureRaw, defaultSecure)
	if !ok {
		slog.Warn("invalid COOKIE_SECURE value, falling back to default", "value", cookieSecureRaw, "default", secureCookie)
	}

	// CORS許可オリジン（デフォルト: http://localhost:3000）
	corsOrigins := config.ParseCORSOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if corsOrigins == nil {
		corsOrigins = []string{"http://localhost:3000"}
	}

	oauth, err := loadOAuthConfig()
	if err != nil {
		return serverConfig{}, err
	}

	return serverConfig{
		jwtSecret:      jwtSecret,
		passwordPepper: passwordPepper,
		secureCookie:   secureCookie,
		corsOrigins:    corsOrigins,
		oauth:          oauth,
	}, nil
}

// loadOAuthConfig は OAuth 関連の環境変数を検証する。
// GOOGLE_CLIENT_ID / GITHUB_CLIENT_ID のいずれも未設定なら OAuth 無効として nil を返す。
func loadOAuthConfig() (*oauthConfig, error) {
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	if googleClientID == "" && githubClientID == "" {
		return nil, nil
	}

	frontendURL := os.Getenv("OAUTH_FRONTEND_REDIRECT_URL")
	if frontendURL == "" {
		return nil, fmt.Errorf("OAUTH_FRONTEND_REDIRECT_URL is required when OAuth is enabled")
	}

	cfg := &oauthConfig{frontendURL: frontendURL}

	if googleClientID != "" {
		secret := os.Getenv("GOOGLE_CLIENT_SECRET")
		if secret == "" {
			return nil, fmt.Errorf("GOOGLE_CLIENT_SECRET is required when GOOGLE_CLIENT_ID is set")
		}
		redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
		if redirectURL == "" {
			return nil, fmt.Errorf("GOOGLE_REDIRECT_URL is required when GOOGLE_CLIENT_ID is set")
		}
		cfg.google = &oauthProviderConfig{clientID: googleClientID, clientSecret: secret, redirectURL: redirectURL}
	}

	if githubClientID != "" {
		secret := os.Getenv("GITHUB_CLIENT_SECRET")
		if secret == "" {
			return nil, fmt.Errorf("GITHUB_CLIENT_SECRET is required when GITHUB_CLIENT_ID is set")
		}
		redirectURL := os.Getenv("GITHUB_REDIRECT_URL")
		if redirectURL == "" {
			return nil, fmt.Errorf("GITHUB_REDIRECT_URL is required when GITHUB_CLIENT_ID is set")
		}
		cfg.github = &oauthProviderConfig{clientID: githubClientID, clientSecret: secret, redirectURL: redirectURL}
	}

	return cfg, nil
}

// main は run の戻り値で os.Exit するだけのラッパー。
// os.Exit は defer を実行しないため、DB / Redis / Vision クライアントの
// Close 等の後処理が走るよう実体は run に分離している。
func main() {
	os.Exit(run())
}

// run は API サーバーを構成・起動し、終了コードを返す。
// 設定不正は 2、外部接続や起動の失敗は 1、正常終了は 0。
func run() int {
	// 構造化ロガーを初期化
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "DEBUG" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// 環境変数を外部接続前に検証する
	cfg, err := loadServerConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		return 2
	}

	// データベース接続。スキーマ適用は cmd/migrate バイナリ（goose）で別途実施する。
	sqlDB, err := infradb.OpenSQL()
	if err != nil {
		slog.Error("DB open failed", "error", err)
		return 1
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			slog.Warn("failed to close sqlDB", "error", err)
		}
	}()

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

	// 全 feature が sqlc 化済み。
	userRepo := authadapters.NewUserRepository(sqlDB)
	symbolRepo := symbollistadapters.NewSymbolRepository(sqlDB)
	candleRepo := candles.NewCandleRepository(sqlDB)
	watchlistRepo := watchlist.NewWatchlistRepository(sqlDB)

	// Redisキャッシュでラップ（TTLはingest連続失敗時のセーフティネット、通常は日次ingestで上書き）
	cachedCandleRepo := candles.NewCachingCandleRepository(rdb, candles.DefaultCacheTTL, candleRepo, "candles")

	// JWTジェネレータ
	jwtGen := jwtmw.NewGenerator(cfg.jwtSecret, 1*time.Hour)

	// Google Cloudクライアント初期化
	visionDetector, err := logovision.NewVisionLogoDetector(context.Background())
	if err != nil {
		slog.Error("failed to create vision client", "error", err)
		return 1
	}
	defer func() {
		if err := visionDetector.Close(); err != nil {
			slog.Error("Failed to close vision client", "error", err)
		}
	}()

	geminiAnalyzer, err := logogemini.NewGeminiAnalyzer(context.Background())
	if err != nil {
		slog.Error("failed to create gemini client", "error", err)
		return 1
	}

	// レートリミッター
	rateLimiter := httpratelimit.NewLimiter(rdb)

	// ユースケース
	authUC := authusecase.NewAuthUsecase(userRepo, jwtGen, cfg.passwordPepper)
	symbolUC := symbollistusecase.NewSymbolUsecase(symbolRepo)
	candlesUC := candles.NewCandlesUsecase(cachedCandleRepo)
	logoUC := logousecase.NewLogoDetectionUsecase(visionDetector, geminiAnalyzer)
	watchlistUC := watchlist.NewWatchlistUsecase(watchlistRepo, symbolRepo)

	// OAuth ハンドラー（cfg.oauth が nil の場合はOAuth機能なしで起動）
	var oauthH *authhandler.OAuthHandler
	if cfg.oauth != nil {
		if rdb == nil {
			slog.Error("OAuth requires Redis but Redis is unavailable")
			return 1
		}
		oauthProviders := map[string]authusecase.OAuthProvider{}
		if cfg.oauth.google != nil {
			oauthProviders["google"] = authadapters.NewGoogleProvider(
				cfg.oauth.google.clientID,
				cfg.oauth.google.clientSecret,
				cfg.oauth.google.redirectURL,
				&http.Client{Timeout: 10 * time.Second},
			)
		}
		if cfg.oauth.github != nil {
			oauthProviders["github"] = authadapters.NewGitHubProvider(
				cfg.oauth.github.clientID,
				cfg.oauth.github.clientSecret,
				cfg.oauth.github.redirectURL,
				&http.Client{Timeout: 10 * time.Second},
			)
		}
		oauthUC := authusecase.NewOAuthUsecase(
			userRepo,
			authadapters.NewOAuthAccountRepository(sqlDB),
			userRepo,
			authadapters.NewRedisOAuthStateStore(rdb),
			jwtGen,
			oauthProviders,
			watchlistUC,
		)
		oauthH = authhandler.NewOAuthHandler(oauthUC, cfg.secureCookie, cfg.oauth.frontendURL)
	}

	// ハンドラー
	authH := authhandler.NewAuthHandler(authUC, rateLimiter, cfg.secureCookie, watchlistUC)
	symbolH := symbollisthandler.NewSymbolHandler(symbolUC)
	candlesH := candleshttp.NewCandlesHandler(candlesUC)
	logoH := logohandler.NewLogoDetectionHandler(logoUC)
	watchlistH := watchlisthttp.NewWatchlistHandler(watchlistUC)

	// ルーター作成
	r := router.NewRouter(authH, oauthH, candlesH, symbolH, logoH, watchlistH, rateLimiter, cfg.corsOrigins)

	slog.Info("Starting server", "port", 8080)
	if err := r.Run(":8080"); err != nil {
		slog.Error("Server failed to start", "error", err)
		return 1
	}
	return 0
}
