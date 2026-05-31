package jwtmw

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
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

// TestAuthRequired_MissingBearerToken はBearerトークンがない場合やプレフィックスが不正な場合に401が返されることを検証します。
func TestAuthRequired_MissingBearerToken(t *testing.T) {
	// Set up environment for this test
	t.Setenv(EnvKeyJWTSecret, "test-secret")

	tests := []struct {
		name       string
		authHeader string
	}{
		{"no header", ""},
		{"empty header", ""},
		{"basic auth", "Basic dXNlcjpwYXNz"},
		{"bearer lowercase", "bearer token123"},
		{"no space after Bearer", "Bearertoken123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				c.Request.Header.Set("Authorization", tt.authHeader)
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
	// Ensure JWT_SECRET is not set (t.Setenv with empty string)
	t.Setenv(EnvKeyJWTSecret, "")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("Authorization", "Bearer sometoken")

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
			c.Request.Header.Set("Authorization", "Bearer "+tt.token)

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
		userID         int64
		expectedUserID int64
	}{
		{"user id 1", 1, 1},
		{"user id 42", 42, 42},
		{"max int64 user id", 9223372036854775807, 9223372036854775807},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := createTokenWithSecret(testSecret, tt.userID, time.Hour)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Request.Header.Set("Authorization", "Bearer "+token)

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
			if userID.(int64) != tt.expectedUserID {
				t.Errorf("expected userID %d, got %d", tt.expectedUserID, userID)
			}
		})
	}
}

// TestAuthRequired_LegacyNumericSubject は移行前の数値subjectが安全な範囲で受理されることを検証します。
func TestAuthRequired_LegacyNumericSubject(t *testing.T) {
	const testSecret = "test-secret-key-for-legacy"
	t.Setenv(EnvKeyJWTSecret, testSecret)

	token := createLegacyTokenWithSecret(testSecret, 42, time.Hour)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	AuthRequired()(c)

	if c.IsAborted() {
		t.Fatalf("expected request not to be aborted, response: %s", w.Body.String())
	}
	if userID, _ := c.Get(ContextUserID); userID != int64(42) {
		t.Errorf("expected userID 42, got %v", userID)
	}
}

// TestAuthRequired_InvalidSubject は不正なsubjectが拒否されることを検証します。
func TestAuthRequired_InvalidSubject(t *testing.T) {
	const testSecret = "test-secret-key-for-subject"
	t.Setenv(EnvKeyJWTSecret, testSecret)

	tests := []struct {
		name string
		sub  any
	}{
		{"missing subject", nil},
		{"non-numeric string", "abc"},
		{"zero", "0"},
		{"unsafe legacy number", float64(1 << 53)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := createTokenWithSubject(testSecret, tt.sub, time.Hour)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Request.Header.Set("Authorization", "Bearer "+token)

			AuthRequired()(c)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
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
	c.Request.Header.Set("Authorization", "Bearer "+tokenStr)

	handler := AuthRequired()
	handler(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// createTokenWithSecret はテスト用に指定されたシークレットとユーザーIDで署名済みJWTトークンを生成します。
func createTokenWithSecret(secret string, userID int64, expiration time.Duration) string {
	return createTokenWithSubject(secret, strconv.FormatInt(userID, 10), expiration)
}

// createLegacyTokenWithSecret は移行前と同じ数値subjectのトークンを生成します。
func createLegacyTokenWithSecret(secret string, userID int64, expiration time.Duration) string {
	return createTokenWithSubject(secret, float64(userID), expiration)
}

func createTokenWithSubject(secret string, subject any, expiration time.Duration) string {
	claims := jwt.MapClaims{
		"sub":   subject,
		"exp":   time.Now().Add(expiration).Unix(),
		"iat":   time.Now().Unix(),
		"email": "test@example.com",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}
