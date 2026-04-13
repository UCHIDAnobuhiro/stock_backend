// Package middleware はプラットフォーム共通のHTTPミドルウェアを提供します。
package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders はセキュリティ関連のHTTPレスポンスヘッダーを設定するGinミドルウェアを返します。
// このAPIサーバーはJSONのみを返すため、CSPは最も制限的な設定を使用します。
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'none'")
		c.Next()
	}
}
