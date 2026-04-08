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

// ContextAuthSource はGinコンテキストに認証方式（"cookie" または "bearer"）を格納するためのキーです。
// CSRFミドルウェアがBearer認証の場合にCSRFチェックをスキップするために使用します。
const ContextAuthSource = "auth_source"

// AuthRequired はJWTトークンを検証し、認証済みユーザーのみにアクセスを制限するGinミドルウェアを返します。
// 認証はCookie（auth_token）を優先し、存在しない場合はAuthorizationヘッダーにフォールバックします。
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. auth_token Cookie を優先（Next.jsブラウザクライアント用）
		var tokenStr string
		if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
			tokenStr = cookie
			c.Set(ContextAuthSource, "cookie")
		} else {
			// 2. Authorization: Bearer ヘッダーにフォールバック（APIクライアント・curl等）
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				tokenStr = strings.TrimPrefix(auth, "Bearer ")
				c.Set(ContextAuthSource, "bearer")
			}
		}
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authentication token"})
			return
		}

		// 3. 環境変数からシークレットキーを読み込み
		secret := os.Getenv(EnvKeyJWTSecret)
		if secret == "" {
			// サーバー設定ミス（JWT_SECRETが未設定）
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server misconfigured"})
			return
		}

		// 4. JWT署名をパースして検証
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

		// 5. クレーム（ペイロード）を抽出
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}
		sub, ok := claims["sub"].(float64) // JWTの数値はfloat64としてデコードされる
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token: missing subject"})
			return
		}
		c.Set(ContextUserID, uint(sub))

		// 6. 次のハンドラーに制御を渡す
		c.Next()
	}
}
