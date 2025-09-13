package infrastructure

import (
	jwtmw "stock_backend/internal/infrastructure/jwt"
	"stock_backend/internal/interface/handler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter(authHandler *handler.AuthHandler, candles *handler.CandlesHandler,
	symbol *handler.SymbolHandler) *gin.Engine {
	r := gin.Default()
	// CORS のデフォルト設定を有効
	r.Use(cors.Default())

	// 認証不要
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
		auth.GET("/healthz", handler.Health)
		auth.GET("/candles/:code", candles.GetCandlesHandler)
		auth.GET("/symbols", symbol.List)
	}

	// TODO:開発用。不要になったら削除
	// r.GET("/healthz", handler.Health)
	// r.GET("/candles/:code", candles.GetCandlesHandler)
	// r.GET("/symbols", symbol.List)

	return r
}
