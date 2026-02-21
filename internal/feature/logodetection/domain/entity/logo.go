// Package entity はlogodetectionフィーチャーのドメインモデルを定義します。
package entity

// DetectedLogo は画像から検出されたロゴを表します。
type DetectedLogo struct {
	Name       string  // 検出された企業名
	Confidence float32 // 信頼度スコア（0.0 ~ 1.0）
}
