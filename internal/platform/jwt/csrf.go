package jwtmw

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CSRFRequired はDouble Submit Cookieパターンによるクロスサイトリクエストフォージェリ対策ミドルウェアを返します。
// csrf_token CookieとX-CSRF-Tokenヘッダーの値を比較し、一致しない場合は403を返します。
func CSRFRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookieToken, err := c.Cookie(CookieCSRFToken)
		if err != nil || cookieToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "csrf token missing"})
			return
		}

		headerToken := c.GetHeader(HeaderCSRFToken)
		if headerToken == "" || headerToken != cookieToken {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "csrf token mismatch"})
			return
		}

		c.Next()
	}
}

// GenerateCSRFToken はcrypto/randを使用して安全なランダムCSRFトークンを生成します。
func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
