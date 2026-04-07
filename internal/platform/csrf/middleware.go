// Package csrf はDouble Submit CookieパターンによるCSRF保護を提供します。
package csrf

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"

	"stock_backend/internal/api"
)

const (
	// CookieName はCSRFトークンを格納するCookie名です。
	// このCookieはhttpOnly=falseのため、JavaScriptから読み取り可能です。
	CookieName = "csrf_token"

	// HeaderName はCSRFトークンを送信するリクエストヘッダー名です。
	HeaderName = "X-CSRF-Token"

	tokenBytes = 32
)

// GenerateToken は暗号学的に安全な64文字のhex文字列（CSRFトークン）を生成します。
func GenerateToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Protect はDouble Submit CookieパターンでCSRF攻撃を防ぐGinミドルウェアを返します。
//   - GET / HEAD / OPTIONS などの安全なメソッドはスキップします
//   - それ以外のメソッドでは X-CSRF-Token ヘッダーと csrf_token Cookie の値が
//     一致しない場合に 403 を返します
func Protect() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		cookieVal, err := c.Cookie(CookieName)
		if err != nil || cookieVal == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, api.ErrorResponse{Error: "missing csrf token"})
			return
		}

		headerVal := c.GetHeader(HeaderName)
		if headerVal == "" || headerVal != cookieVal {
			c.AbortWithStatusJSON(http.StatusForbidden, api.ErrorResponse{Error: "csrf token mismatch"})
			return
		}

		c.Next()
	}
}
