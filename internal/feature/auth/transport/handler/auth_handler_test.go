package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAuthUsecase is a mock implementation of the usecase.AuthUsecase interface.
type mockAuthUsecase struct {
	SignupFunc func(ctx context.Context, email, password string) error
	LoginFunc  func(ctx context.Context, email, password string) (string, error)
}

// Signup is the mock implementation of the Signup method.
func (m *mockAuthUsecase) Signup(ctx context.Context, email, password string) error {
	if m.SignupFunc != nil {
		return m.SignupFunc(ctx, email, password)
	}
	return nil // Default: success
}

// Login is the mock implementation of the Login method.
func (m *mockAuthUsecase) Login(ctx context.Context, email, password string) (string, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, email, password)
	}
	return "", errors.New("login failed") // Default: failure
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
		mockLoginFunc  func(ctx context.Context, email, password string) (string, error)
		expectedStatus int
		expectedBody   gin.H
	}{
		{
			name:           "success: user login",
			requestBody:    gin.H{"email": "test@example.com", "password": "password123"},
			mockLoginFunc:  func(ctx context.Context, email, password string) (string, error) { return "dummy-jwt-token", nil },
			expectedStatus: http.StatusOK,
			expectedBody:   gin.H{"token": "dummy-jwt-token"},
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
			name:           "failure: invalid credentials (usecase error)",
			requestBody:    gin.H{"email": "wrong@example.com", "password": "wrong-password"},
			mockLoginFunc:  func(ctx context.Context, email, password string) (string, error) { return "", errors.New("invalid email or password") },
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
			expectedBody:   gin.H{"error": "invalid email or password"}, // Usecase error message is hidden
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
