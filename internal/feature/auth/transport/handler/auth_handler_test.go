package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"stock_backend/internal/feature/auth/usecase"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAuthUsecase is a mock implementation of the usecase.AuthUsecase interface.
type mockAuthUsecase struct {
	SignupFunc       func(ctx context.Context, email, password string) error
	LoginFunc        func(ctx context.Context, email, password, userAgent, ipAddress string) (*usecase.LoginResult, error)
	RefreshTokenFunc func(ctx context.Context, refreshToken, userAgent, ipAddress string) (*usecase.RefreshResult, error)
	LogoutFunc       func(ctx context.Context, refreshToken string) error
}

// Signup is the mock implementation of the Signup method.
func (m *mockAuthUsecase) Signup(ctx context.Context, email, password string) error {
	if m.SignupFunc != nil {
		return m.SignupFunc(ctx, email, password)
	}
	return nil // Default: success
}

// Login is the mock implementation of the Login method.
func (m *mockAuthUsecase) Login(ctx context.Context, email, password, userAgent, ipAddress string) (*usecase.LoginResult, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, email, password, userAgent, ipAddress)
	}
	return nil, errors.New("login failed") // Default: failure
}

// RefreshToken is the mock implementation of the RefreshToken method.
func (m *mockAuthUsecase) RefreshToken(ctx context.Context, refreshToken, userAgent, ipAddress string) (*usecase.RefreshResult, error) {
	if m.RefreshTokenFunc != nil {
		return m.RefreshTokenFunc(ctx, refreshToken, userAgent, ipAddress)
	}
	return nil, errors.New("refresh failed") // Default: failure
}

// Logout is the mock implementation of the Logout method.
func (m *mockAuthUsecase) Logout(ctx context.Context, refreshToken string) error {
	if m.LogoutFunc != nil {
		return m.LogoutFunc(ctx, refreshToken)
	}
	return nil // Default: success
}

// TestMain sets up the test environment once for all tests.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

// makeRequest is a helper function to create and execute an HTTP request.
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

// assertJSONResponse is a helper function to validate JSON response status and body.
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedBody gin.H) {
	t.Helper()

	assert.Equal(t, expectedStatus, w.Code)

	var responseBody gin.H
	err := json.Unmarshal(w.Body.Bytes(), &responseBody)
	require.NoError(t, err)

	assert.Equal(t, expectedBody, responseBody)
}

func TestAuthHandler_Signup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		requestBody    gin.H
		mockSignupFunc func(ctx context.Context, email, password string) error
		expectedStatus int
		expectedBody   gin.H
	}{
		{
			name:           "success: user registration",
			requestBody:    gin.H{"email": "test@example.com", "password": "password123"},
			mockSignupFunc: func(ctx context.Context, email, password string) error { return nil },
			expectedStatus: http.StatusCreated,
			expectedBody:   gin.H{"message": "ok"},
		},
		{
			name:           "failure: invalid email address",
			requestBody:    gin.H{"email": "invalid-email", "password": "password123"},
			mockSignupFunc: nil, // Usecase is not called
			expectedStatus: http.StatusBadRequest,
			expectedBody:   gin.H{"error": "invalid request"},
		},
		{
			name:           "failure: short password",
			requestBody:    gin.H{"email": "test@example.com", "password": "short"},
			mockSignupFunc: nil, // Usecase is not called
			expectedStatus: http.StatusBadRequest,
			expectedBody:   gin.H{"error": "invalid request"},
		},
		{
			name:           "failure: duplicate email (usecase error)",
			requestBody:    gin.H{"email": "existing@example.com", "password": "password123"},
			mockSignupFunc: func(ctx context.Context, email, password string) error { return errors.New("email already exists") },
			expectedStatus: http.StatusConflict,
			expectedBody:   gin.H{"error": "signup failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUC := &mockAuthUsecase{SignupFunc: tt.mockSignupFunc}
			handler := NewAuthHandler(mockUC)

			router := gin.New()
			router.POST("/signup", handler.Signup)

			w := makeRequest(t, router, http.MethodPost, "/signup", tt.requestBody)
			assertJSONResponse(t, w, tt.expectedStatus, tt.expectedBody)
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		requestBody    gin.H
		mockLoginFunc  func(ctx context.Context, email, password, userAgent, ipAddress string) (*usecase.LoginResult, error)
		expectedStatus int
		expectedBody   gin.H
	}{
		{
			name:        "success: user login",
			requestBody: gin.H{"email": "test@example.com", "password": "password123"},
			mockLoginFunc: func(ctx context.Context, email, password, userAgent, ipAddress string) (*usecase.LoginResult, error) {
				return &usecase.LoginResult{
					AccessToken:  "dummy-access-token",
					RefreshToken: "dummy-refresh-token",
					ExpiresIn:    900,
					TokenType:    "Bearer",
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody: gin.H{
				"access_token":  "dummy-access-token",
				"refresh_token": "dummy-refresh-token",
				"expires_in":    float64(900),
				"token_type":    "Bearer",
			},
		},
		{
			name:           "failure: invalid email address",
			requestBody:    gin.H{"email": "invalid-email", "password": "password123"},
			mockLoginFunc:  nil, // Usecase is not called
			expectedStatus: http.StatusBadRequest,
			expectedBody:   gin.H{"error": "invalid request"},
		},
		{
			name:           "failure: missing password",
			requestBody:    gin.H{"email": "test@example.com"},
			mockLoginFunc:  nil, // Usecase is not called
			expectedStatus: http.StatusBadRequest,
			expectedBody:   gin.H{"error": "invalid request"},
		},
		{
			name:        "failure: invalid credentials (usecase error)",
			requestBody: gin.H{"email": "wrong@example.com", "password": "wrong-password"},
			mockLoginFunc: func(ctx context.Context, email, password, userAgent, ipAddress string) (*usecase.LoginResult, error) {
				return nil, errors.New("invalid email or password")
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   gin.H{"error": "invalid email or password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUC := &mockAuthUsecase{LoginFunc: tt.mockLoginFunc}
			handler := NewAuthHandler(mockUC)

			router := gin.New()
			router.POST("/login", handler.Login)

			w := makeRequest(t, router, http.MethodPost, "/login", tt.requestBody)
			assertJSONResponse(t, w, tt.expectedStatus, tt.expectedBody)
		})
	}
}

func TestAuthHandler_Refresh(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		requestBody        gin.H
		mockRefreshFunc    func(ctx context.Context, refreshToken, userAgent, ipAddress string) (*usecase.RefreshResult, error)
		expectedStatus     int
		expectedBody       gin.H
	}{
		{
			name:        "success: token refresh",
			requestBody: gin.H{"refresh_token": "valid-refresh-token"},
			mockRefreshFunc: func(ctx context.Context, refreshToken, userAgent, ipAddress string) (*usecase.RefreshResult, error) {
				return &usecase.RefreshResult{
					AccessToken:  "new-access-token",
					RefreshToken: "new-refresh-token",
					ExpiresIn:    900,
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody: gin.H{
				"access_token":  "new-access-token",
				"refresh_token": "new-refresh-token",
				"expires_in":    float64(900),
			},
		},
		{
			name:            "failure: missing refresh_token",
			requestBody:     gin.H{},
			mockRefreshFunc: nil, // Usecase is not called
			expectedStatus:  http.StatusBadRequest,
			expectedBody:    gin.H{"error": "invalid request"},
		},
		{
			name:        "failure: invalid refresh token",
			requestBody: gin.H{"refresh_token": "invalid-token"},
			mockRefreshFunc: func(ctx context.Context, refreshToken, userAgent, ipAddress string) (*usecase.RefreshResult, error) {
				return nil, errors.New("invalid refresh token")
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   gin.H{"error": "invalid refresh token"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUC := &mockAuthUsecase{RefreshTokenFunc: tt.mockRefreshFunc}
			handler := NewAuthHandler(mockUC)

			router := gin.New()
			router.POST("/refresh", handler.Refresh)

			w := makeRequest(t, router, http.MethodPost, "/refresh", tt.requestBody)
			assertJSONResponse(t, w, tt.expectedStatus, tt.expectedBody)
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		requestBody    gin.H
		mockLogoutFunc func(ctx context.Context, refreshToken string) error
		expectedStatus int
		expectedBody   gin.H
	}{
		{
			name:        "success: logout",
			requestBody: gin.H{"refresh_token": "valid-refresh-token"},
			mockLogoutFunc: func(ctx context.Context, refreshToken string) error {
				return nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   gin.H{"message": "logged out"},
		},
		{
			name:           "failure: missing refresh_token",
			requestBody:    gin.H{},
			mockLogoutFunc: nil, // Usecase is not called
			expectedStatus: http.StatusBadRequest,
			expectedBody:   gin.H{"error": "invalid request"},
		},
		{
			name:        "success: logout even if token already invalid",
			requestBody: gin.H{"refresh_token": "invalid-token"},
			mockLogoutFunc: func(ctx context.Context, refreshToken string) error {
				return errors.New("session not found")
			},
			expectedStatus: http.StatusOK, // Always return 200 to prevent token enumeration
			expectedBody:   gin.H{"message": "logged out"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUC := &mockAuthUsecase{LogoutFunc: tt.mockLogoutFunc}
			handler := NewAuthHandler(mockUC)

			router := gin.New()
			router.POST("/logout", handler.Logout)

			w := makeRequest(t, router, http.MethodPost, "/logout", tt.requestBody)
			assertJSONResponse(t, w, tt.expectedStatus, tt.expectedBody)
		})
	}
}
