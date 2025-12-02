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

func TestAuthHandler_Signup(t *testing.T) {
	gin.SetMode(gin.TestMode)

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
			expectedBody:   gin.H{"error": "Key: 'SignupReq.Email' Error:Field validation for 'Email' failed on the 'email' tag"},
		},
		{
			name:           "failure: short password",
			requestBody:    gin.H{"email": "test@example.com", "password": "short"},
			mockSignupFunc: nil, // Usecase is not called
			expectedStatus: http.StatusBadRequest,
			expectedBody:   gin.H{"error": "Key: 'SignupReq.Password' Error:Field validation for 'Password' failed on the 'min' tag"},
		},
		{
			name:           "failure: duplicate email (usecase error)",
			requestBody:    gin.H{"email": "existing@example.com", "password": "password123"},
			mockSignupFunc: func(ctx context.Context, email, password string) error { return errors.New("email already exists") },
			expectedStatus: http.StatusConflict,
			expectedBody:   gin.H{"error": "email already exists"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &mockAuthUsecase{SignupFunc: tt.mockSignupFunc}
			handler := NewAuthHandler(mockUC)

			router := gin.New()
			router.POST("/signup", handler.Signup)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var responseBody gin.H
			err := json.Unmarshal(w.Body.Bytes(), &responseBody)
			assert.NoError(t, err)

			// Error messages include Gin validation error details, so check partial match
			if tt.expectedStatus == http.StatusBadRequest {
				assert.Contains(t, responseBody["error"], tt.expectedBody["error"])
			} else {
				assert.Equal(t, tt.expectedBody, responseBody)
			}
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	gin.SetMode(gin.TestMode)

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
			expectedBody:   gin.H{"error": "Key: 'LoginReq.Email' Error:Field validation for 'Email' failed on the 'email' tag"},
		},
		{
			name:           "failure: missing password",
			requestBody:    gin.H{"email": "test@example.com"},
			mockLoginFunc:  nil, // Usecase is not called
			expectedStatus: http.StatusBadRequest,
			expectedBody:   gin.H{"error": "Key: 'LoginReq.Password' Error:Field validation for 'Password' failed on the 'required' tag"},
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
			mockUC := &mockAuthUsecase{LoginFunc: tt.mockLoginFunc}
			handler := NewAuthHandler(mockUC)

			router := gin.New()
			router.POST("/login", handler.Login)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var responseBody gin.H
			err := json.Unmarshal(w.Body.Bytes(), &responseBody)
			assert.NoError(t, err)

			// Error messages include Gin validation error details, so check partial match
			if tt.expectedStatus == http.StatusBadRequest {
				assert.Contains(t, responseBody["error"], tt.expectedBody["error"])
			} else {
				assert.Equal(t, tt.expectedBody, responseBody)
			}
		})
	}
}
