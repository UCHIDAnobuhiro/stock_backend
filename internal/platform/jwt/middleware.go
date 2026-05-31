package jwtmw

import (
	"errors"
	"math"
	"net/http"
	"os"
	"strconv"
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
		userID, err := parseSubject(claims["sub"])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token: invalid subject"})
			return
		}
		c.Set(ContextUserID, userID)

		// 6. 次のハンドラーに制御を渡す
		c.Next()
	}
}

// parseSubject はJWT subjectをユーザーIDへ変換します。
// 新規トークンは文字列を使用しますが、移行中の既存トークン向けに安全な範囲の数値も受理します。
func parseSubject(claim any) (int64, error) {
	switch sub := claim.(type) {
	case string:
		userID, err := strconv.ParseInt(sub, 10, 64)
		if err != nil || userID <= 0 {
			return 0, errors.New("subject must be a positive integer")
		}
		return userID, nil
	case float64:
		const maxSafeInteger = float64(1<<53 - 1)
		if sub <= 0 || sub > maxSafeInteger || math.Trunc(sub) != sub {
			return 0, errors.New("numeric subject must be a safe positive integer")
		}
		return int64(sub), nil
	default:
		return 0, errors.New("subject must be a string or number")
	}
}
