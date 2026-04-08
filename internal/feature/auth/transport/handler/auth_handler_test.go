package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"stock_backend/internal/feature/auth/transport/handler"
	"stock_backend/internal/platform/ratelimit"
)

// mockAuthUsecase はAuthUsecaseインターフェースのモック実装です。
type mockAuthUsecase struct {
	SignupFunc func(ctx context.Context, email, password string) (uint, error)
	LoginFunc  func(ctx context.Context, email, password string) (string, error)
}

// Signup はSignupメソッドのモック実装です。
func (m *mockAuthUsecase) Signup(ctx context.Context, email, password string) (uint, error) {
	if m.SignupFunc != nil {
		return m.SignupFunc(ctx, email, password)
	}
	return 1, nil // デフォルト: 成功
}

// Login はLoginメソッドのモック実装です。
func (m *mockAuthUsecase) Login(ctx context.Context, email, password string) (string, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, email, password)
	}
	return "", errors.New("login failed") // デフォルト: 失敗
}

// TestMain は全テスト共通のテスト環境を設定します。
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

// makeRequest はHTTPリクエストを作成・実行するヘルパー関数です。
func makeRequest(t *testing.T, router *gin.Engine, method, path string, body gin.H) *httptest.ResponseRecorder {
	t.Helper()

	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest(method, path, bytes.NewBuffer(bodyBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

// assertJSONResponse はJSONレスポンスのステータスコードとボディを検証するヘルパー関数です。
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedBody gin.H) {
	t.Helper()

	assert.Equal(t, expectedStatus, w.Code)

	var responseBody gin.H
	err := json.Unmarshal(w.Body.Bytes(), &responseBody)
	require.NoError(t, err)

	assert.Equal(t, expectedBody, responseBody)
}

// assertLoginCookies はログイン成功時のSet-CookieヘッダーにCookieが正しく設定されていることを検証します。
func assertLoginCookies(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	var authTokenCookie, csrfTokenCookie string
	for _, c := range w.Header().Values("Set-Cookie") {
		if strings.HasPrefix(c, "auth_token=") {
			authTokenCookie = c
		}
		if strings.HasPrefix(c, "csrf_token=") {
			csrfTokenCookie = c
		}
	}

	// auth_token: HttpOnly かつ SameSite=Lax であること
	assert.NotEmpty(t, authTokenCookie, "auth_token cookie should be set")
	assert.Contains(t, authTokenCookie, "HttpOnly", "auth_token should be HttpOnly")
	assert.Contains(t, authTokenCookie, "SameSite=Lax", "auth_token should have SameSite=Lax")

	// csrf_token: 非HttpOnly（JavaScriptから読み取れる）かつ SameSite=Lax であること
	assert.NotEmpty(t, csrfTokenCookie, "csrf_token cookie should be set")
	assert.NotContains(t, csrfTokenCookie, "HttpOnly", "csrf_token must not be HttpOnly")
	assert.Contains(t, csrfTokenCookie, "SameSite=Lax", "csrf_token should have SameSite=Lax")
}

// TestAuthHandler_Signup はサインアップハンドラーのHTTPリクエスト/レスポンス処理をテストします。
func TestAuthHandler_Signup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		requestBody    gin.H
		mockSignupFunc func(ctx context.Context, email, password string) (uint, error)
		expectedStatus int
		expectedBody   gin.H
	}{
		{
			name:           "success: user registration",
			requestBody:    gin.H{"email": "test@example.com", "password": "password123"},
			mockSignupFunc: func(ctx context.Context, email, password string) (uint, error) { return 1, nil },
			expectedStatus: http.StatusCreated,
			expectedBody:   gin.H{"message": "ok"},
		},
		{
			name:           "failure: invalid email address",
			requestBody:    gin.H{"email": "invalid-email", "password": "password123"},
			mockSignupFunc: nil, // Usecaseは呼ばれない
			expectedStatus: http.StatusBadRequest,
			expectedBody:   gin.H{"error": "invalid request"},
		},
		{
			name:           "failure: short password",
			requestBody:    gin.H{"email": "test@example.com", "password": "short"},
			mockSignupFunc: nil, // Usecaseは呼ばれない
			expectedStatus: http.StatusBadRequest,
			expectedBody:   gin.H{"error": "invalid request"},
		},
		{
			name:        "failure: duplicate email (usecase error)",
			requestBody: gin.H{"email": "existing@example.com", "password": "password123"},
			mockSignupFunc: func(ctx context.Context, email, password string) (uint, error) {
				return 0, errors.New("email already exists")
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   gin.H{"error": "signup failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUC := &mockAuthUsecase{SignupFunc: tt.mockSignupFunc}
			h := handler.NewAuthHandler(mockUC, nil, false)

			router := gin.New()
			router.POST("/signup", h.Signup)

			w := makeRequest(t, router, http.MethodPost, "/signup", tt.requestBody)
			assertJSONResponse(t, w, tt.expectedStatus, tt.expectedBody)
		})
	}
}

// TestAuthHandler_Login_RateLimited はメールベースのレートリミット超過時に429が返されることを検証します。
func TestAuthHandler_Login_RateLimited(t *testing.T) {
	t.Parallel()

	rdb, mock := redismock.NewClientMock()
	t.Cleanup(func() { _ = rdb.Close() })

	// Luaスクリプトモック: allowed=0（レートリミット超過）を返す
	match := mock.CustomMatch(func(expected, actual []interface{}) error {
		return nil
	})
	key := "rl:login:email:test@example.com"
	match.ExpectEvalSha(ratelimit.ScriptHash(), []string{key},
		"_", "_", "_", "_", "_").
		SetVal([]interface{}{int64(0), int64(5)})

	limiter := ratelimit.NewLimiter(rdb)
	loginCalled := false
	mockUC := &mockAuthUsecase{
		LoginFunc: func(ctx context.Context, email, password string) (string, error) {
			loginCalled = true
			return "", errors.New("should not be called")
		},
	}
	h := handler.NewAuthHandler(mockUC, limiter, false)

	router := gin.New()
	router.POST("/login", h.Login)

	w := makeRequest(t, router, http.MethodPost, "/login", gin.H{
		"email":    "test@example.com",
		"password": "password123",
	})

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "900", w.Header().Get("Retry-After"))

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "too many requests", body["error"])

	assert.False(t, loginCalled, "レートリミット超過時はUsecaseが呼ばれないこと")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAuthHandler_Login はログインハンドラーのHTTPリクエスト/レスポンス処理をテストします。
func TestAuthHandler_Login(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		requestBody    gin.H
		mockLoginFunc  func(ctx context.Context, email, password string) (string, error)
		expectedStatus int
		expectedBody   gin.H
		checkCookies   bool
	}{
		{
			name:           "success: user login",
			requestBody:    gin.H{"email": "test@example.com", "password": "password123"},
			mockLoginFunc:  func(ctx context.Context, email, password string) (string, error) { return "dummy-jwt-token", nil },
			expectedStatus: http.StatusOK,
			expectedBody:   gin.H{"message": "ok"},
			checkCookies:   true,
		},
		{
			name:           "failure: invalid email address",
			requestBody:    gin.H{"email": "invalid-email", "password": "password123"},
			mockLoginFunc:  nil, // Usecaseは呼ばれない
			expectedStatus: http.StatusBadRequest,
			expectedBody:   gin.H{"error": "invalid request"},
		},
		{
			name:           "failure: missing password",
			requestBody:    gin.H{"email": "test@example.com"},
			mockLoginFunc:  nil, // Usecaseは呼ばれない
			expectedStatus: http.StatusBadRequest,
			expectedBody:   gin.H{"error": "invalid request"},
		},
		{
			name:        "failure: invalid credentials (usecase error)",
			requestBody: gin.H{"email": "wrong@example.com", "password": "wrong-password"},
			mockLoginFunc: func(ctx context.Context, email, password string) (string, error) {
				return "", errors.New("invalid email or password")
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   gin.H{"error": "invalid email or password"},
		},
		{
			name:        "failure: JWT secret not set (usecase error)",
			requestBody: gin.H{"email": "test@example.com", "password": "password123"},
			mockLoginFunc: func(ctx context.Context, email, password string) (string, error) {
				return "", errors.New("server misconfigured: JWT_SECRET missing")
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   gin.H{"error": "invalid email or password"}, // Usecaseのエラーメッセージは隠蔽される
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUC := &mockAuthUsecase{LoginFunc: tt.mockLoginFunc}
			h := handler.NewAuthHandler(mockUC, nil, false)

			router := gin.New()
			router.POST("/login", h.Login)

			w := makeRequest(t, router, http.MethodPost, "/login", tt.requestBody)
			assertJSONResponse(t, w, tt.expectedStatus, tt.expectedBody)
			if tt.checkCookies {
				assertLoginCookies(t, w)
			}
		})
	}
}

// TestAuthHandler_Logout はログアウトハンドラーがCookieを削除することを検証します。
func TestAuthHandler_Logout(t *testing.T) {
	t.Parallel()

	h := handler.NewAuthHandler(&mockAuthUsecase{}, nil, false)

	router := gin.New()
	router.POST("/logout", h.Logout)

	w := makeRequest(t, router, http.MethodPost, "/logout", gin.H{})

	assert.Equal(t, http.StatusOK, w.Code)

	var authTokenCookie, csrfTokenCookie string
	for _, c := range w.Header().Values("Set-Cookie") {
		if strings.HasPrefix(c, "auth_token=") {
			authTokenCookie = c
		}
		if strings.HasPrefix(c, "csrf_token=") {
			csrfTokenCookie = c
		}
	}

	// ログアウト時は Max-Age=0 でCookieを削除すること
	assert.NotEmpty(t, authTokenCookie, "auth_token cookie should be present in response")
	assert.Contains(t, authTokenCookie, "Max-Age=0", "auth_token cookie should be deleted (Max-Age=0)")

	assert.NotEmpty(t, csrfTokenCookie, "csrf_token cookie should be present in response")
	assert.Contains(t, csrfTokenCookie, "Max-Age=0", "csrf_token cookie should be deleted (Max-Age=0)")
}
