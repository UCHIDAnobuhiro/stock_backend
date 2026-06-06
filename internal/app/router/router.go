// Package router はアプリケーションのHTTPルーティングを設定します。
package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	authhttp "stock_backend/internal/feature/auth/transport"
	candleshttp "stock_backend/internal/feature/candles/transport"
	logodetectionhttp "stock_backend/internal/feature/logodetection/transport"
	symbollisthttp "stock_backend/internal/feature/symbollist/transport"
	watchlisthttp "stock_backend/internal/feature/watchlist/transport"
	csrfmw "stock_backend/internal/platform/csrf"
	handler "stock_backend/internal/platform/handler"
	"stock_backend/internal/platform/httpratelimit"
	jwtmw "stock_backend/internal/platform/jwt"
	httpmw "stock_backend/internal/platform/middleware"
)

// NewRouter はすべてのアプリケーションルートを設定したGinルーターを生成します。
// 公開ルート（signup, login）とJWT認証ミドルウェア付きの保護ルート（candles, symbols, logo, watchlist）を設定します。
// oauthHandler が nil の場合はOAuthルートを登録しません。
func NewRouter(authHandler *authhttp.AuthHandler, oauthHandler *authhttp.OAuthHandler,
	candles *candleshttp.CandlesHandler,
	symbol *symbollisthttp.SymbolHandler, logo *logodetectionhttp.LogoDetectionHandler,
	watchlist *watchlisthttp.WatchlistHandler,
	limiter *httpratelimit.Limiter,
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
			httpratelimit.ByIP(limiter, httpratelimit.IPRateLimitConfig{
				Prefix: "rl:signup:ip",
				Limit:  5,
				Window: 1 * time.Hour,
			}),
			authHandler.Signup,
		)
		v1.POST("/login",
			httpratelimit.ByIP(limiter, httpratelimit.IPRateLimitConfig{
				Prefix: "rl:login:ip",
				Limit:  10,
				Window: 1 * time.Minute,
			}),
			authHandler.Login,
		)
		// 期限切れトークンでもログアウトできるよう認証不要
		v1.DELETE("/logout", authHandler.Logout)

		// OAuthルート（環境変数が設定されている場合のみ登録）
		if oauthHandler != nil {
			oauthGroup := v1.Group("/auth/oauth")
			oauthGroup.GET("/:provider", oauthHandler.BeginAuth)
			oauthGroup.GET("/:provider/callback",
				httpratelimit.ByIP(limiter, httpratelimit.IPRateLimitConfig{
					Prefix: "rl:oauth:callback:ip",
					Limit:  20,
					Window: 1 * time.Minute,
				}),
				oauthHandler.Callback,
			)
		}

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
