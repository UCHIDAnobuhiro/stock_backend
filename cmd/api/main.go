package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	redisv9 "github.com/redis/go-redis/v9"

	"github.com/UCHIDAnobuhiro/stock-backend/internal/app/config"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/app/di"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/app/router"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/auth"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/auth/authhttp"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/candles"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/candles/candleshttp"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/logodetection"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/logodetection/gemini"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/logodetection/logodetectionhttp"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/logodetection/vision"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/symbollist"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/symbollist/symbollisthttp"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/watchlist"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/watchlist/watchlisthttp"
	infradb "github.com/UCHIDAnobuhiro/stock-backend/internal/infra/db"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/infra/logging"
	infraredis "github.com/UCHIDAnobuhiro/stock-backend/internal/infra/redis"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/transport/httpratelimit"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/transport/jwt"
)

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
	useJSON, formatOK := config.ParseLogFormat(os.Getenv("LOG_FORMAT"), os.Getenv("APP_ENV"))
	logger := slog.New(logging.NewHandler(os.Stdout, logLevel, useJSON))
	slog.SetDefault(logger)
	if !formatOK {
		slog.Warn("invalid LOG_FORMAT value, using default", "value", os.Getenv("LOG_FORMAT"))
	}

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
	userRepo := auth.NewUserRepository(sqlDB)
	symbolRepo := symbollist.NewRepository(sqlDB)
	candleRepo := candles.NewRepository(sqlDB)
	watchlistRepo := watchlist.NewRepository(sqlDB)

	// Redisキャッシュでラップ（TTLはingest連続失敗時のセーフティネット、通常は日次ingestで上書き）
	cachedCandleRepo := candles.NewCachingRepository(rdb, candles.DefaultCacheTTL, candleRepo, "candles")

	// JWTジェネレータ
	jwtGen := jwt.NewGenerator(cfg.jwtSecret, 1*time.Hour)

	// Google Cloudクライアント初期化
	visionDetector, err := vision.NewVisionLogoDetector(context.Background())
	if err != nil {
		slog.Error("failed to create vision client", "error", err)
		return 1
	}
	defer func() {
		if err := visionDetector.Close(); err != nil {
			slog.Error("Failed to close vision client", "error", err)
		}
	}()

	geminiAnalyzer, err := gemini.NewGeminiAnalyzer(context.Background())
	if err != nil {
		slog.Error("failed to create gemini client", "error", err)
		return 1
	}

	// レートリミッター
	rateLimiter := httpratelimit.NewLimiter(rdb)

	// ユースケース
	authUC := auth.NewUsecase(userRepo, jwtGen, cfg.passwordPepper)
	symbolUC := symbollist.NewUsecase(symbolRepo)
	candlesUC := candles.NewUsecase(cachedCandleRepo)
	logoUC := logodetection.NewUsecase(visionDetector, geminiAnalyzer)
	watchlistUC := watchlist.NewUsecase(watchlistRepo, symbolRepo)

	// OAuth ハンドラー（cfg.oauth が nil の場合はOAuth機能なしで起動）
	var oauthH *authhttp.OAuthHandler
	if cfg.oauth != nil {
		oauthH, err = di.NewOAuthHandler(cfg.oauth, sqlDB, rdb, userRepo, jwtGen, watchlistUC, cfg.secureCookie)
		if err != nil {
			slog.Error("failed to set up OAuth", "error", err)
			return 1
		}
	}

	// ハンドラー
	authH := authhttp.NewHandler(authUC, rateLimiter, cfg.secureCookie, watchlistUC)
	symbolH := symbollisthttp.NewHandler(symbolUC)
	candlesH := candleshttp.NewHandler(candlesUC)
	logoH := logodetectionhttp.NewHandler(logoUC)
	watchlistH := watchlisthttp.NewHandler(watchlistUC)

	// ルーター作成
	r := router.NewRouter(authH, oauthH, candlesH, symbolH, logoH, watchlistH, rateLimiter, cfg.corsOrigins, cfg.gcpProjectID)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// SIGINT / SIGTERM を受けてグレースフルシャットダウンする。
	// Cloud Run 等では SIGTERM 受信後に処理中リクエストを完了させてから終了する。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("Starting server", "port", 8080)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			slog.Error("Server failed to start", "error", err)
			return 1
		}
		return 0
	case <-ctx.Done():
		slog.Info("Shutdown signal received, draining connections")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("graceful shutdown failed", "error", err)
			return 1
		}
		slog.Info("Server stopped gracefully")
		return 0
	}
}

// serverConfig は環境変数から読み込んだ検証済みのサーバー設定。
type serverConfig struct {
	jwtSecret      string
	passwordPepper string
	secureCookie   bool
	corsOrigins    []string
	gcpProjectID   string          // GOOGLE_CLOUD_PROJECT。未設定可（トレース相関に使用）
	oauth          *di.OAuthConfig // OAuth 無効なら nil
}

// loadServerConfig は環境変数を読み込んで検証し、検証済みの設定を返す。
// 外部接続（DB/Redis/GCP）に依存しない純粋関数なのでユニットテスト可能。
// 必須項目の欠落や OAuth 設定の不整合があればエラーを返す。
func loadServerConfig() (serverConfig, error) {
	jwtSecret := os.Getenv(jwt.EnvKeyJWTSecret)
	if jwtSecret == "" {
		return serverConfig{}, fmt.Errorf("%s is required", jwt.EnvKeyJWTSecret)
	}

	passwordPepper := os.Getenv(auth.EnvKeyPasswordPepper)
	if passwordPepper == "" {
		return serverConfig{}, fmt.Errorf("%s is required", auth.EnvKeyPasswordPepper)
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
		gcpProjectID:   os.Getenv("GOOGLE_CLOUD_PROJECT"),
		oauth:          oauth,
	}, nil
}

// loadOAuthConfig は OAuth 関連の環境変数を検証する。
// GOOGLE_CLIENT_ID / GITHUB_CLIENT_ID のいずれも未設定なら OAuth 無効として nil を返す。
func loadOAuthConfig() (*di.OAuthConfig, error) {
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	if googleClientID == "" && githubClientID == "" {
		return nil, nil
	}

	frontendURL := os.Getenv("OAUTH_FRONTEND_REDIRECT_URL")
	if frontendURL == "" {
		return nil, fmt.Errorf("OAUTH_FRONTEND_REDIRECT_URL is required when OAuth is enabled")
	}

	cfg := &di.OAuthConfig{FrontendURL: frontendURL}

	if googleClientID != "" {
		secret := os.Getenv("GOOGLE_CLIENT_SECRET")
		if secret == "" {
			return nil, fmt.Errorf("GOOGLE_CLIENT_SECRET is required when GOOGLE_CLIENT_ID is set")
		}
		redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
		if redirectURL == "" {
			return nil, fmt.Errorf("GOOGLE_REDIRECT_URL is required when GOOGLE_CLIENT_ID is set")
		}
		cfg.Google = &di.ProviderCredentials{ClientID: googleClientID, ClientSecret: secret, RedirectURL: redirectURL}
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
		cfg.GitHub = &di.ProviderCredentials{ClientID: githubClientID, ClientSecret: secret, RedirectURL: redirectURL}
	}

	return cfg, nil
}
