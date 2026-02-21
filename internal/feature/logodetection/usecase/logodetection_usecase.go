// Package usecase はlogodetectionフィーチャーのビジネスロジックを実装します。
package usecase

import (
	"context"
	"fmt"
	"regexp"
	"unicode/utf8"

	"stock_backend/internal/feature/logodetection/domain/entity"
)

const (
	// MaxImageSize は画像アップロードの最大サイズ（10MB）です。
	MaxImageSize = 10 * 1024 * 1024
	// AnalysisPromptTemplate は企業分析のプロンプトテンプレートです。
	AnalysisPromptTemplate = "日本語で、企業分析の観点から%sの強みを3つ挙げて。"
	// MaxCompanyNameLength は企業名の最大文字数（rune数）です。
	MaxCompanyNameLength = 100
)

// validCompanyName は企業名に許可される文字パターンです（英数字・日本語・スペース・中黒）。
var validCompanyName = regexp.MustCompile(`^[\p{L}\p{N}\s・\-\.&,]+$`)

// LogoDetector は画像からロゴを検出するリポジトリインターフェースです。
// Goの慣例に従い、インターフェースは利用者（usecase）側で定義します。
type LogoDetector interface {
	// DetectLogos は画像バイト列からロゴを検出し、検出結果を返します。
	DetectLogos(ctx context.Context, imageData []byte) ([]entity.DetectedLogo, error)
}

// CompanyAnalyzer は企業分析を生成するリポジトリインターフェースです。
// Goの慣例に従い、インターフェースは利用者（usecase）側で定義します。
type CompanyAnalyzer interface {
	// Analyze はプロンプトから分析サマリーを生成します。
	Analyze(ctx context.Context, prompt string) (string, error)
}

// logodetectionUsecase はロゴ検出・企業分析のビジネスロジックを提供します。
type logodetectionUsecase struct {
	logoDetector    LogoDetector
	companyAnalyzer CompanyAnalyzer
}

// NewLogoDetectionUsecase はlogodetectionUsecaseの新しいインスタンスを生成します。
func NewLogoDetectionUsecase(ld LogoDetector, ca CompanyAnalyzer) *logodetectionUsecase {
	return &logodetectionUsecase{logoDetector: ld, companyAnalyzer: ca}
}

// DetectLogos は画像データからロゴを検出します。
func (u *logodetectionUsecase) DetectLogos(ctx context.Context, imageData []byte) ([]entity.DetectedLogo, error) {
	if len(imageData) == 0 {
		return nil, fmt.Errorf("image data is empty")
	}
	if len(imageData) > MaxImageSize {
		return nil, fmt.Errorf("image size exceeds maximum of %d bytes", MaxImageSize)
	}
	return u.logoDetector.DetectLogos(ctx, imageData)
}

// AnalyzeCompany は企業名から分析サマリーを生成します。
func (u *logodetectionUsecase) AnalyzeCompany(ctx context.Context, companyName string) (*entity.CompanyAnalysis, error) {
	if companyName == "" {
		return nil, fmt.Errorf("company name is required")
	}
	if utf8.RuneCountInString(companyName) > MaxCompanyNameLength {
		return nil, fmt.Errorf("company name exceeds maximum length of %d characters", MaxCompanyNameLength)
	}
	if !validCompanyName.MatchString(companyName) {
		return nil, fmt.Errorf("company name contains invalid characters")
	}
	prompt := fmt.Sprintf(AnalysisPromptTemplate, companyName)
	summary, err := u.companyAnalyzer.Analyze(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("company analyzer failed for %q: %w", companyName, err)
	}
	return &entity.CompanyAnalysis{
		CompanyName: companyName,
		Summary:     summary,
	}, nil
}
