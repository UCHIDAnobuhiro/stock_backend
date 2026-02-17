// Package gemini はGoogle Gemini APIを使用した企業分析クライアントを提供します。
package gemini

import (
	"context"
	"fmt"

	"google.golang.org/genai"

	"stock_backend/internal/feature/logodetection/usecase"
)

const (
	// DefaultModel はGemini APIのデフォルトモデルです。
	DefaultModel = "gemini-2.5-flash"
)

// GeminiAnalyzer はGoogle Gemini APIを使用して企業分析を生成します。
type GeminiAnalyzer struct {
	client *genai.Client
	model  string
}

// GeminiAnalyzerがCompanyAnalyzerを実装していることをコンパイル時に検証します。
var _ usecase.CompanyAnalyzer = (*GeminiAnalyzer)(nil)

// NewGeminiAnalyzer はADCを使用してGeminiAnalyzerの新しいインスタンスを生成します。
// 環境変数 GOOGLE_GENAI_USE_VERTEXAI, GOOGLE_CLOUD_PROJECT, GOOGLE_CLOUD_LOCATION が必要です。
func NewGeminiAnalyzer(ctx context.Context) (*GeminiAnalyzer, error) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}
	return &GeminiAnalyzer{client: client, model: DefaultModel}, nil
}

// Analyze はプロンプトを使用して分析サマリーを生成します。
func (g *GeminiAnalyzer) Analyze(ctx context.Context, prompt string) (string, error) {
	resp, err := g.client.Models.GenerateContent(ctx, g.model, genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("gemini API request failed: %w", err)
	}

	return resp.Text(), nil
}
