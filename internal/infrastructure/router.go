package infrastructure

import (
	jwtmw "todo_backend/internal/infrastructure/jwt"
	"todo_backend/internal/interface/handler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter(authHandler *handler.AuthHandler) *gin.Engine {
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
	}

	return r
}
