package jwtmw

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ContextUserID はGinコンテキストに認証済みユーザーIDを格納するためのキーです。
const ContextUserID = "userID"

// AuthRequired はJWTトークンを検証し、認証済みユーザーのみにアクセスを制限するGinミドルウェアを返します。
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Authorizationヘッダーを取得
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		// 2. 環境変数からシークレットキーを読み込み
		secret := os.Getenv(EnvKeyJWTSecret)
		if secret == "" {
			// サーバー設定ミス（JWT_SECRETが未設定）
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server misconfigured"})
			return
		}

		// 3. JWT署名をパースして検証
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			// 署名アルゴリズムを確認（HMACのみ許可）
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			// 検証エラーまたは無効なトークン
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// 4. クレーム（ペイロード）を抽出
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if sub, ok := claims["sub"].(float64); ok { // JWTの数値はfloat64としてデコードされる
				c.Set(ContextUserID, uint(sub))
			}
		}
		// 5. 次のハンドラーに制御を渡す
		c.Next()
	}
}
