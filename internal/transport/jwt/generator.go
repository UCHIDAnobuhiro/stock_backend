// Package jwt はJWTトークンの生成と認証ミドルウェアを提供します。
package jwt

import (
	"fmt"
	"strconv"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// Generator はJWTトークンの生成を実装します。
// 利用者（例: auth/usecase）が定義するJWTGeneratorインターフェースを実装します。
type Generator struct {
	secret     []byte
	expiration time.Duration
}

// NewGenerator は指定されたシークレットと有効期限でJWTジェネレータの新しいインスタンスを生成します。
func NewGenerator(secret string, expiration time.Duration) *Generator {
	return &Generator{
		secret:     []byte(secret),
		expiration: expiration,
	}
}

// GenerateToken は標準クレームを含む署名済みJWTトークンを生成します。
func (g *Generator) GenerateToken(userID int64, email string) (string, error) {
	claims := gojwt.MapClaims{
		"sub":   strconv.FormatInt(userID, 10),
		"exp":   time.Now().Add(g.expiration).Unix(),
		"iat":   time.Now().Unix(),
		"email": email,
	}

	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(g.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signed, nil
}
