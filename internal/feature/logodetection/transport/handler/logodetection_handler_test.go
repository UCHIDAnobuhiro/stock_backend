package handler_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"stock_backend/internal/feature/logodetection/domain/entity"
	"stock_backend/internal/feature/logodetection/transport/handler"
)

// mockLogoDetectionUsecase はLogoDetectionUsecaseインターフェースのモック実装です。
type mockLogoDetectionUsecase struct {
	DetectLogosFunc    func(ctx context.Context, imageData []byte) ([]entity.DetectedLogo, error)
	AnalyzeCompanyFunc func(ctx context.Context, companyName string) (*entity.CompanyAnalysis, error)
}

func (m *mockLogoDetectionUsecase) DetectLogos(ctx context.Context, imageData []byte) ([]entity.DetectedLogo, error) {
	return m.DetectLogosFunc(ctx, imageData)
}

func (m *mockLogoDetectionUsecase) AnalyzeCompany(ctx context.Context, companyName string) (*entity.CompanyAnalysis, error) {
	return m.AnalyzeCompanyFunc(ctx, companyName)
}

// createMultipartRequest はテスト用のマルチパートリクエストを生成するヘルパー関数です。
func createMultipartRequest(t *testing.T, fieldName, fileName string, content []byte) (*http.Request, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}

	if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
		t.Fatalf("failed to copy content: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/logo/detect", body)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, writer.FormDataContentType()
}

func TestLogoDetectionHandler_DetectLogos(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		mockFunc       func(ctx context.Context, imageData []byte) ([]entity.DetectedLogo, error)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success: logos detected",
			setupRequest: func(t *testing.T) *http.Request {
				req, _ := createMultipartRequest(t, "image", "test.jpg", []byte("fake-image"))
				return req
			},
			mockFunc: func(ctx context.Context, imageData []byte) ([]entity.DetectedLogo, error) {
				return []entity.DetectedLogo{
					{Name: "Apple", Confidence: 0.95},
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"name":"Apple","confidence":0.95}]`,
		},
		{
			name: "error: no image field",
			setupRequest: func(t *testing.T) *http.Request {
				req, _ := http.NewRequest(http.MethodPost, "/logo/detect", io.NopCloser(bytes.NewReader(nil)))
				return req
			},
			mockFunc:       nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"画像ファイルが必要です"}`,
		},
		{
			name: "error: usecase returns error",
			setupRequest: func(t *testing.T) *http.Request {
				req, _ := createMultipartRequest(t, "image", "test.jpg", []byte("fake-image"))
				return req
			},
			mockFunc: func(ctx context.Context, imageData []byte) ([]entity.DetectedLogo, error) {
				return nil, errors.New("vision API error")
			},
			expectedStatus: http.StatusBadGateway,
			expectedBody:   `{"error":"ロゴ検出に失敗しました"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &mockLogoDetectionUsecase{
				DetectLogosFunc: tt.mockFunc,
			}

			h := handler.NewLogoDetectionHandler(mockUC)

			router := gin.New()
			router.POST("/logo/detect", h.DetectLogos)

			w := httptest.NewRecorder()
			req := tt.setupRequest(t)

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestLogoDetectionHandler_AnalyzeCompany(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    string
		mockFunc       func(ctx context.Context, companyName string) (*entity.CompanyAnalysis, error)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "success: analysis generated",
			requestBody: `{"company_name":"任天堂"}`,
			mockFunc: func(ctx context.Context, companyName string) (*entity.CompanyAnalysis, error) {
				assert.Equal(t, "任天堂", companyName)
				return &entity.CompanyAnalysis{
					CompanyName: "任天堂",
					Summary:     "任天堂の強みは...",
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"company_name":"任天堂","summary":"任天堂の強みは..."}`,
		},
		{
			name:           "error: empty request body",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"企業名が必要です"}`,
		},
		{
			name:           "error: invalid json",
			requestBody:    `invalid`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"企業名が必要です"}`,
		},
		{
			name:        "error: usecase returns error",
			requestBody: `{"company_name":"テスト企業"}`,
			mockFunc: func(ctx context.Context, companyName string) (*entity.CompanyAnalysis, error) {
				return nil, errors.New("gemini API error")
			},
			expectedStatus: http.StatusBadGateway,
			expectedBody:   `{"error":"企業分析に失敗しました"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &mockLogoDetectionUsecase{
				AnalyzeCompanyFunc: tt.mockFunc,
			}

			h := handler.NewLogoDetectionHandler(mockUC)

			router := gin.New()
			router.POST("/logo/analyze", h.AnalyzeCompany)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/logo/analyze", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}
