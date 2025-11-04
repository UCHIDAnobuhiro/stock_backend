package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// mockAuthUsecase は usecase.AuthUsecase インターフェースのモック実装です。
type mockAuthUsecase struct {
	SignupFunc func(email, password string) error
	LoginFunc  func(email, password string) (string, error)
}

// Signup はモックの Signup メソッドです。
func (m *mockAuthUsecase) Signup(email, password string) error {
	if m.SignupFunc != nil {
		return m.SignupFunc(email, password)
	}
	return nil // デフォルトでは成功
}

// Login はモックの Login メソッドです。
func (m *mockAuthUsecase) Login(email, password string) (string, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(email, password)
	}
	return "", errors.New("login failed") // デフォルトでは失敗
}

func TestAuthHandler_Signup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    gin.H
		mockSignupFunc func(email, password string) error
		expectedStatus int
		expectedBody   gin.H
	}{
		{
			name:           "成功: ユーザー登録",
			requestBody:    gin.H{"email": "test@example.com", "password": "password123"},
			mockSignupFunc: func(email, password string) error { return nil },
			expectedStatus: http.StatusCreated,
			expectedBody:   gin.H{"message": "ok"},
		},
		{
			name:           "失敗: 無効なメールアドレス",
			requestBody:    gin.H{"email": "invalid-email", "password": "password123"},
			mockSignupFunc: nil, // Usecaseは呼ばれない
			expectedStatus: http.StatusBadRequest,
			expectedBody:   gin.H{"error": "Key: 'SignupReq.Email' Error:Field validation for 'Email' failed on the 'email' tag"},
		},
		{
			name:           "失敗: 短いパスワード",
			requestBody:    gin.H{"email": "test@example.com", "password": "short"},
			mockSignupFunc: nil, // Usecaseは呼ばれない
			expectedStatus: http.StatusBadRequest,
			expectedBody:   gin.H{"error": "Key: 'SignupReq.Password' Error:Field validation for 'Password' failed on the 'min' tag"},
		},
		{
			name:           "失敗: メールアドレス重複 (Usecaseエラー)",
			requestBody:    gin.H{"email": "existing@example.com", "password": "password123"},
			mockSignupFunc: func(email, password string) error { return errors.New("email already exists") },
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

			// エラーメッセージはGinのバリデーションエラーの詳細を含むため、部分一致で確認
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
		mockLoginFunc  func(email, password string) (string, error)
		expectedStatus int
		expectedBody   gin.H
	}{
		{
			name:           "成功: ユーザーログイン",
			requestBody:    gin.H{"email": "test@example.com", "password": "password123"},
			mockLoginFunc:  func(email, password string) (string, error) { return "dummy-jwt-token", nil },
			expectedStatus: http.StatusOK,
			expectedBody:   gin.H{"token": "dummy-jwt-token"},
		},
		{
			name:           "失敗: 無効なメールアドレス",
			requestBody:    gin.H{"email": "invalid-email", "password": "password123"},
			mockLoginFunc:  nil, // Usecaseは呼ばれない
			expectedStatus: http.StatusBadRequest,
			expectedBody:   gin.H{"error": "Key: 'loginReq.Email' Error:Field validation for 'Email' failed on the 'email' tag"},
		},
		{
			name:           "失敗: パスワードなし",
			requestBody:    gin.H{"email": "test@example.com"},
			mockLoginFunc:  nil, // Usecaseは呼ばれない
			expectedStatus: http.StatusBadRequest,
			expectedBody:   gin.H{"error": "Key: 'loginReq.Password' Error:Field validation for 'Password' failed on the 'required' tag"},
		},
		{
			name:           "失敗: 認証情報不正 (Usecaseエラー)",
			requestBody:    gin.H{"email": "wrong@example.com", "password": "wrong-password"},
			mockLoginFunc:  func(email, password string) (string, error) { return "", errors.New("invalid email or password") },
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   gin.H{"error": "invalid email or password"},
		},
		{
			name:        "失敗: JWTシークレット未設定 (Usecaseエラー)",
			requestBody: gin.H{"email": "test@example.com", "password": "password123"},
			mockLoginFunc: func(email, password string) (string, error) {
				return "", errors.New("server misconfigured: JWT_SECRET missing")
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   gin.H{"error": "invalid email or password"}, // Usecaseのエラーメッセージは隠蔽される
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

			// エラーメッセージはGinのバリデーションエラーの詳細を含むため、部分一致で確認
			if tt.expectedStatus == http.StatusBadRequest {
				assert.Contains(t, responseBody["error"], tt.expectedBody["error"])
			} else {
				assert.Equal(t, tt.expectedBody, responseBody)
			}
		})
	}
}
