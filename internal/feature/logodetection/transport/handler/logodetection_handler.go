// Package handler はlogodetectionフィーチャーのHTTPハンドラーを提供します。
package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"stock_backend/internal/api"
	"stock_backend/internal/feature/logodetection/domain/entity"
)

// LogoDetectionUsecase はロゴ検出・企業分析のユースケースインターフェースを定義します。
// Goの慣例に従い、インターフェースは利用者（handler）側で定義します。
type LogoDetectionUsecase interface {
	DetectLogos(ctx context.Context, imageData []byte) ([]entity.DetectedLogo, error)
	AnalyzeCompany(ctx context.Context, companyName string) (*entity.CompanyAnalysis, error)
}

// LogoDetectionHandler はロゴ検出・企業分析のHTTPリクエストを処理します。
type LogoDetectionHandler struct {
	uc LogoDetectionUsecase
}

// NewLogoDetectionHandler はLogoDetectionHandlerの新しいインスタンスを生成します。
func NewLogoDetectionHandler(uc LogoDetectionUsecase) *LogoDetectionHandler {
	return &LogoDetectionHandler{uc: uc}
}

// DetectLogos は画像をアップロードしてロゴを検出します。
//
// エンドポイント: POST /v1/logo/detect
// Content-Type: multipart/form-data
// フィールド: image（画像ファイル、最大10MB）
func (h *LogoDetectionHandler) DetectLogos(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		slog.Warn("画像ファイルの取得に失敗", "error", err, "remote_addr", c.ClientIP())
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "画像ファイルが必要です"})
		return
	}

	f, err := file.Open()
	if err != nil {
		slog.Error("画像ファイルのオープンに失敗", "error", err)
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: "画像の読み込みに失敗しました"})
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			slog.Warn("画像ファイルのクローズに失敗", "error", err)
		}
	}()

	imageData, err := io.ReadAll(f)
	if err != nil {
		slog.Error("画像データの読み取りに失敗", "error", err)
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: "画像の読み込みに失敗しました"})
		return
	}

	logos, err := h.uc.DetectLogos(c.Request.Context(), imageData)
	if err != nil {
		slog.Error("ロゴ検出に失敗", "error", err)
		c.JSON(http.StatusBadGateway, api.ErrorResponse{Error: "ロゴ検出に失敗しました"})
		return
	}

	out := make([]api.DetectedLogoResponse, 0, len(logos))
	for _, l := range logos {
		out = append(out, api.DetectedLogoResponse{
			Name:       l.Name,
			Confidence: l.Confidence,
		})
	}
	c.JSON(http.StatusOK, out)
}

// AnalyzeCompany は企業分析サマリーを生成します。
//
// エンドポイント: POST /v1/logo/analyze
// Content-Type: application/json
func (h *LogoDetectionHandler) AnalyzeCompany(c *gin.Context) {
	var req api.CompanyAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("企業分析リクエストのバリデーションに失敗", "error", err, "remote_addr", c.ClientIP())
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "企業名が必要です"})
		return
	}

	analysis, err := h.uc.AnalyzeCompany(c.Request.Context(), req.CompanyName)
	if err != nil {
		slog.Error("企業分析に失敗", "error", err, "company", req.CompanyName)
		c.JSON(http.StatusBadGateway, api.ErrorResponse{Error: "企業分析に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, api.CompanyAnalysisResponse{
		CompanyName: analysis.CompanyName,
		Summary:     analysis.Summary,
	})
}
