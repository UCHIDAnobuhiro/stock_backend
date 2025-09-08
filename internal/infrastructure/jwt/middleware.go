package jwtmw

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const ContextUserID = "userID"

// AuthRequiredはJWTを検証し、認証されたユーザーのみがアクセスできるようにする
// Ginのミドルウェア関数を返します
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Authorization ヘッダーの取得
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		// 2. 秘密鍵の読み込み（環境変数から）
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			// サーバー側の設定ミス（JWT_SECRET未設定）
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server misconfigured"})
			return
		}

		// 3. JWT のパースと署名検証
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			// 署名アルゴリズムのチェック（HMACのみ許可）
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			// 検証エラーまたは不正なトークン
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// 4. Claims（ペイロード部分）の取り出し
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if sub, ok := claims["sub"].(float64); ok { // JWTはjsonでfloatになる
				c.Set(ContextUserID, uint(sub))
			}
		}
		// 5. 次のハンドラへ処理を渡す
		c.Next()
	}
}
