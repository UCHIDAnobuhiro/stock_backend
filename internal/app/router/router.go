// Package router はアプリケーションのHTTPルーティングを設定します。
package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	authhandler "stock_backend/internal/feature/auth/transport/handler"
	candleshandler "stock_backend/internal/feature/candles/transport/handler"
	logohandler "stock_backend/internal/feature/logodetection/transport/handler"
	symbollisthandler "stock_backend/internal/feature/symbollist/transport/handler"
	watchlisthandler "stock_backend/internal/feature/watchlist/transport/handler"
	csrfmw "stock_backend/internal/platform/csrf"
	handler "stock_backend/internal/platform/http/handler"
	httpmw "stock_backend/internal/platform/http/middleware"
	jwtmw "stock_backend/internal/platform/jwt"
	"stock_backend/internal/platform/ratelimit"
)

// NewRouter はすべてのアプリケーションルートを設定したGinルーターを生成します。
// 公開ルート（signup, login）とJWT認証ミドルウェア付きの保護ルート（candles, symbols, logo, watchlist）を設定します。
func NewRouter(authHandler *authhandler.AuthHandler, candles *candleshandler.CandlesHandler,
	symbol *symbollisthandler.SymbolHandler, logo *logohandler.LogoDetectionHandler,
	watchlist *watchlisthandler.WatchlistHandler,
	limiter *ratelimit.Limiter,
	allowedOrigins []string,
) *gin.Engine {
	r := gin.Default()

	// リバースプロキシを使用しない構成のため、X-Forwarded-For等のヘッダーを信頼しない
	// c.ClientIP()がRemoteAddr（実際のTCP接続元）を返すようにする
	_ = r.SetTrustedProxies(nil)

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.Use(httpmw.SecurityHeaders())

	// 画像アップロードのサイズ制限を設定（10MB）
	r.MaxMultipartMemory = 10 << 20

	// ヘルスチェックエンドポイント（バージョンなし）
	r.GET("/healthz", handler.Health)

	// API v1 ルート
	v1 := r.Group("/v1")
	{
		// 公開ルート（認証不要）+ レートリミット
		v1.POST("/signup",
			ratelimit.ByIP(limiter, ratelimit.IPRateLimitConfig{
				Prefix: "rl:signup:ip",
				Limit:  5,
				Window: 1 * time.Hour,
			}),
			authHandler.Signup,
		)
		v1.POST("/login",
			ratelimit.ByIP(limiter, ratelimit.IPRateLimitConfig{
				Prefix: "rl:login:ip",
				Limit:  10,
				Window: 1 * time.Minute,
			}),
			authHandler.Login,
		)
		// 期限切れトークンでもログアウトできるよう認証不要
		v1.DELETE("/logout", authHandler.Logout)

		// 保護ルート（認証必須・CSRF保護）
		auth := v1.Group("/")
		auth.Use(jwtmw.AuthRequired())
		auth.Use(csrfmw.Protect())
		{
			auth.GET("/candles/:code", candles.GetCandlesHandler)
			auth.GET("/symbols", symbol.List)
			auth.POST("/logo/detect", logo.DetectLogos)
			auth.POST("/logo/analyze", logo.AnalyzeCompany)
			auth.GET("/watchlist", watchlist.List)
			auth.POST("/watchlist", watchlist.Add)
			auth.DELETE("/watchlist/:code", watchlist.Remove)
			auth.PUT("/watchlist/order", watchlist.Reorder)
		}
	}

	return r
}
