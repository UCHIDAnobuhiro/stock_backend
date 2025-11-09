package router

import (
	jwtmw "stock_backend/internal/infrastructure/jwt"
	"stock_backend/internal/interface/handler"

	"github.com/gin-gonic/gin"
)

func NewRouter(authHandler *handler.AuthHandler, candles *handler.CandlesHandler,
	symbol *handler.SymbolHandler) *gin.Engine {
	r := gin.Default()

	// 認証不要
	// 導通確認用
	r.GET("/healthz", handler.Health)
	// 新規ユーザー登録
	r.POST("/signup", authHandler.Signup)
	// ログイン（JWT 発行）
	r.POST("/login", authHandler.Login)

	// 認証必須のルート
	// r.Group("/") でルートグループを作成
	auth := r.Group("/")
	// jwtmw.AuthRequired() ミドルウェアを適用
	// → リクエストヘッダーに JWT が必要になる
	auth.Use(jwtmw.AuthRequired())
	{
		auth.GET("/candles/:code", candles.GetCandlesHandler)
		auth.GET("/symbols", symbol.List)
	}

	return r
}
