// Package handler はプラットフォームレベルのエンドポイント用HTTPハンドラーを提供します。
package handler

import "github.com/gin-gonic/gin"

// Health はサービスヘルスチェック用の /healthz エンドポイントを処理します。
// HTTPメソッドに応じて適切にレスポンスし、キャッシュを防止します。
func Health(c *gin.Context) {
	// 明示的にキャッシュを防止
	c.Header("Cache-Control", "no-store")

	// すべてのGET/HEAD/OPTIONSリクエストに対して200または204を返す
	switch c.Request.Method {
	case "HEAD":
		c.Status(200)
	case "OPTIONS":
		c.Status(204)
	default:
		c.JSON(200, gin.H{"status": "ok"})
	}
}
