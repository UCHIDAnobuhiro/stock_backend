package usecase_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"stock_backend/internal/feature/logodetection/domain/entity"
	"stock_backend/internal/feature/logodetection/usecase"
)

// ErrAPI はモックと期待値の間で共有されるセンチネルエラーです。
var ErrAPI = errors.New("api error")

// mockLogoDetector はLogoDetectorインターフェースのモック実装です。
type mockLogoDetector struct {
	DetectLogosFunc  func(ctx context.Context, imageData []byte) ([]entity.DetectedLogo, error)
	DetectLogosCalls int
}

func (m *mockLogoDetector) DetectLogos(ctx context.Context, imageData []byte) ([]entity.DetectedLogo, error) {
	m.DetectLogosCalls++
	if m.DetectLogosFunc != nil {
		return m.DetectLogosFunc(ctx, imageData)
	}
	return nil, errors.New("DetectLogosFunc is not implemented")
}

// mockCompanyAnalyzer はCompanyAnalyzerインターフェースのモック実装です。
type mockCompanyAnalyzer struct {
	AnalyzeFunc  func(ctx context.Context, prompt string) (string, error)
	AnalyzeCalls int
}

func (m *mockCompanyAnalyzer) Analyze(ctx context.Context, prompt string) (string, error) {
	m.AnalyzeCalls++
	if m.AnalyzeFunc != nil {
		return m.AnalyzeFunc(ctx, prompt)
	}
	return "", errors.New("AnalyzeFunc is not implemented")
}

func TestLogoDetectionUsecase_DetectLogos(t *testing.T) {
	ctx := context.Background()
	expectedLogos := []entity.DetectedLogo{
		{Name: "Apple", Confidence: 0.95},
		{Name: "Google", Confidence: 0.87},
	}

	testCases := []struct {
		name          string
		imageData     []byte
		mockFunc      func(ctx context.Context, imageData []byte) ([]entity.DetectedLogo, error)
		expectedLogos []entity.DetectedLogo
		expectedErr   string
	}{
		{
			name:      "success: logos detected",
			imageData: []byte("fake-image-data"),
			mockFunc: func(ctx context.Context, imageData []byte) ([]entity.DetectedLogo, error) {
				return expectedLogos, nil
			},
			expectedLogos: expectedLogos,
		},
		{
			name:        "error: empty image data",
			imageData:   []byte{},
			expectedErr: "image data is empty",
		},
		{
			name:        "error: image too large",
			imageData:   make([]byte, usecase.MaxImageSize+1),
			expectedErr: "image size exceeds maximum",
		},
		{
			name:      "error: api returns error",
			imageData: []byte("fake-image-data"),
			mockFunc: func(ctx context.Context, imageData []byte) ([]entity.DetectedLogo, error) {
				return nil, ErrAPI
			},
			expectedLogos: nil,
			expectedErr:   ErrAPI.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			detector := &mockLogoDetector{DetectLogosFunc: tc.mockFunc}
			analyzer := &mockCompanyAnalyzer{}
			uc := usecase.NewLogoDetectionUsecase(detector, analyzer)

			logos, err := uc.DetectLogos(ctx, tc.imageData)

			if tc.expectedErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.expectedErr)
				}
				if !contains(err.Error(), tc.expectedErr) {
					t.Fatalf("expected error containing %q, got %q", tc.expectedErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(logos, tc.expectedLogos) {
				t.Errorf("result mismatch: got %v, want %v", logos, tc.expectedLogos)
			}
		})
	}
}

func TestLogoDetectionUsecase_AnalyzeCompany(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name            string
		companyName     string
		mockFunc        func(ctx context.Context, prompt string) (string, error)
		expectedSummary string
		expectedErr     string
	}{
		{
			name:        "success: analysis generated",
			companyName: "任天堂",
			mockFunc: func(ctx context.Context, prompt string) (string, error) {
				return "任天堂の強みは...", nil
			},
			expectedSummary: "任天堂の強みは...",
		},
		{
			name:        "error: empty company name",
			companyName: "",
			expectedErr: "company name is required",
		},
		{
			name:        "error: api returns error",
			companyName: "任天堂",
			mockFunc: func(ctx context.Context, prompt string) (string, error) {
				return "", ErrAPI
			},
			expectedErr: ErrAPI.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			detector := &mockLogoDetector{}
			analyzer := &mockCompanyAnalyzer{AnalyzeFunc: tc.mockFunc}
			uc := usecase.NewLogoDetectionUsecase(detector, analyzer)

			result, err := uc.AnalyzeCompany(ctx, tc.companyName)

			if tc.expectedErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.expectedErr)
				}
				if !contains(err.Error(), tc.expectedErr) {
					t.Fatalf("expected error containing %q, got %q", tc.expectedErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.CompanyName != tc.companyName {
				t.Errorf("company name mismatch: got %q, want %q", result.CompanyName, tc.companyName)
			}
			if result.Summary != tc.expectedSummary {
				t.Errorf("summary mismatch: got %q, want %q", result.Summary, tc.expectedSummary)
			}
		})
	}
}

// contains はsがsubstrを含むかどうかを返すヘルパー関数です。
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
