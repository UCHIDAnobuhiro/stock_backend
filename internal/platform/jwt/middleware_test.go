package jwtmw

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// TestMain はテスト実行前にGinをテストモードに設定します。
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// setAuthCookie はテスト用にauth_token Cookieをリクエストにセットするヘルパーです。
func setAuthCookie(req *http.Request, token string) {
	req.AddCookie(&http.Cookie{
		Name:  CookieAuthToken,
		Value: token,
	})
}

// TestAuthRequired_MissingCookie はauth_token Cookieが存在しない場合に401が返されることを検証します。
func TestAuthRequired_MissingCookie(t *testing.T) {
	t.Setenv(EnvKeyJWTSecret, "test-secret")

	tests := []struct {
		name string
		// Cookieなし、または空のCookie
		setCookie bool
		value     string
	}{
		{"no cookie", false, ""},
		{"empty cookie value", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.setCookie {
				c.Request.AddCookie(&http.Cookie{Name: CookieAuthToken, Value: tt.value})
			}

			handler := AuthRequired()
			handler(c)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
			}
			if !c.IsAborted() {
				t.Error("expected request to be aborted")
			}
		})
	}
}

// TestAuthRequired_MissingJWTSecret はJWT_SECRET環境変数が未設定の場合に500が返されることを検証します。
func TestAuthRequired_MissingJWTSecret(t *testing.T) {
	t.Setenv(EnvKeyJWTSecret, "")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	setAuthCookie(c.Request, "sometoken")

	handler := AuthRequired()
	handler(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// TestAuthRequired_InvalidToken は不正なトークン（改ざん・期限切れ等）で401が返されることを検証します。
func TestAuthRequired_InvalidToken(t *testing.T) {
	const testSecret = "test-secret-key-for-invalid"
	t.Setenv(EnvKeyJWTSecret, testSecret)

	tests := []struct {
		name  string
		token string
	}{
		{"malformed token", "not.a.valid.token"},
		{"random string", "randomstring"},
		{"wrong secret", createTokenWithSecret("wrong-secret", 1, time.Hour)},
		{"expired token", createTokenWithSecret(testSecret, 1, -time.Hour)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			setAuthCookie(c.Request, tt.token)

			handler := AuthRequired()
			handler(c)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
			}
		})
	}
}

// TestAuthRequired_ValidToken は有効なトークンでリクエストが通過し、コンテキストにユーザーIDが設定されることを検証します。
func TestAuthRequired_ValidToken(t *testing.T) {
	const testSecret = "test-secret-key-for-valid"
	t.Setenv(EnvKeyJWTSecret, testSecret)

	tests := []struct {
		name           string
		userID         uint
		expectedUserID uint
	}{
		{"user id 1", 1, 1},
		{"user id 42", 42, 42},
		{"user id 999", 999, 999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := createTokenWithSecret(testSecret, tt.userID, time.Hour)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			setAuthCookie(c.Request, token)

			handler := AuthRequired()
			handler(c)

			if c.IsAborted() {
				t.Errorf("expected request not to be aborted, response: %s", w.Body.String())
				return
			}

			userID, exists := c.Get(ContextUserID)
			if !exists {
				t.Error("expected userID to be set in context")
				return
			}
			if userID.(uint) != tt.expectedUserID {
				t.Errorf("expected userID %d, got %d", tt.expectedUserID, userID)
			}
		})
	}
}

// TestAuthRequired_InvalidSigningMethod はnoneアルゴリズム（未署名）のトークンが拒否されることを検証します。
func TestAuthRequired_InvalidSigningMethod(t *testing.T) {
	const testSecret = "test-secret-key-for-signing"
	t.Setenv(EnvKeyJWTSecret, testSecret)

	// Create token with "none" algorithm (unsigned)
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": float64(1),
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})
	tokenStr, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	setAuthCookie(c.Request, tokenStr)

	handler := AuthRequired()
	handler(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// createTokenWithSecret はテスト用に指定されたシークレットとユーザーIDで署名済みJWTトークンを生成します。
func createTokenWithSecret(secret string, userID uint, expiration time.Duration) string {
	claims := jwt.MapClaims{
		"sub":   float64(userID),
		"exp":   time.Now().Add(expiration).Unix(),
		"iat":   time.Now().Unix(),
		"email": fmt.Sprintf("user%d@example.com", userID),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}
