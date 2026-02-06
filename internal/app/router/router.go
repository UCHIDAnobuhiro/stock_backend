// Package router はアプリケーションのHTTPルーティングを設定します。
package router

import (
	authhandler "stock_backend/internal/feature/auth/transport/handler"
	candleshandler "stock_backend/internal/feature/candles/transport/handler"
	symbollisthandler "stock_backend/internal/feature/symbollist/transport/handler"
	handler "stock_backend/internal/platform/http/handler"
	jwtmw "stock_backend/internal/platform/jwt"

	"github.com/gin-gonic/gin"
)

// NewRouter はすべてのアプリケーションルートを設定したGinルーターを生成します。
// 公開ルート（signup, login）とJWT認証ミドルウェア付きの保護ルート（candles, symbols）を設定します。
func NewRouter(authHandler *authhandler.AuthHandler, candles *candleshandler.CandlesHandler,
	symbol *symbollisthandler.SymbolHandler) *gin.Engine {
	r := gin.Default()

	// ヘルスチェックエンドポイント（バージョンなし）
	r.GET("/healthz", handler.Health)

	// API v1 ルート
	v1 := r.Group("/v1")
	{
		// 公開ルート（認証不要）
		v1.POST("/signup", authHandler.Signup)
		v1.POST("/login", authHandler.Login)

		// 保護ルート（認証必須）
		auth := v1.Group("/")
		auth.Use(jwtmw.AuthRequired())
		{
			auth.GET("/candles/:code", candles.GetCandlesHandler)
			auth.GET("/symbols", symbol.List)
		}
	}

	return r
}
