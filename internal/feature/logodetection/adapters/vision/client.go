// Package vision はGoogle Cloud Vision APIを使用したロゴ検出クライアントを提供します。
package vision

import (
	"context"
	"fmt"

	gvision "cloud.google.com/go/vision/v2/apiv1"
	visionpb "cloud.google.com/go/vision/v2/apiv1/visionpb"

	"stock_backend/internal/feature/logodetection/domain/entity"
	"stock_backend/internal/feature/logodetection/usecase"
)

// VisionLogoDetector はGoogle Cloud Vision APIを使用してロゴを検出します。
type VisionLogoDetector struct {
	client *gvision.ImageAnnotatorClient
}

// VisionLogoDetectorがLogoDetectorを実装していることをコンパイル時に検証します。
var _ usecase.LogoDetector = (*VisionLogoDetector)(nil)

// NewVisionLogoDetector はADCを使用してVisionLogoDetectorの新しいインスタンスを生成します。
func NewVisionLogoDetector(ctx context.Context) (*VisionLogoDetector, error) {
	client, err := gvision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create vision client: %w", err)
	}
	return &VisionLogoDetector{client: client}, nil
}

// Close はVision APIクライアントを解放します。
func (v *VisionLogoDetector) Close() error {
	return v.client.Close()
}

// DetectLogos は画像バイト列からロゴを検出します。
func (v *VisionLogoDetector) DetectLogos(ctx context.Context, imageData []byte) ([]entity.DetectedLogo, error) {
	req := &visionpb.BatchAnnotateImagesRequest{
		Requests: []*visionpb.AnnotateImageRequest{
			{
				Image: &visionpb.Image{Content: imageData},
				Features: []*visionpb.Feature{
					{Type: visionpb.Feature_LOGO_DETECTION},
				},
			},
		},
	}

	resp, err := v.client.BatchAnnotateImages(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vision API request failed: %w", err)
	}

	if len(resp.Responses) == 0 {
		return nil, nil
	}

	if resp.Responses[0].Error != nil {
		return nil, fmt.Errorf("vision API error: %s", resp.Responses[0].Error.Message)
	}

	logos := make([]entity.DetectedLogo, 0, len(resp.Responses[0].LogoAnnotations))
	for _, logo := range resp.Responses[0].LogoAnnotations {
		logos = append(logos, entity.DetectedLogo{
			Name:       logo.Description,
			Confidence: logo.Score,
		})
	}

	return logos, nil
}
