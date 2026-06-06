// Package twelvedata はTwelve Data株式市場APIのクライアントを提供します。
package twelvedata

import (
	"os"
	"time"
)

// Config はTwelve Data APIクライアントの設定を保持します。
type Config struct {
	TwelveDataAPIKey string        // 認証用APIキー
	BaseURL          string        // APIのベースURL（例: "https://api.twelvedata.com"）
	Timeout          time.Duration // HTTPリクエストタイムアウト

	// リトライ設定（5xx・ネットワークエラー・429 を対象とする指数バックオフ）。
	MaxRetries       int           // リトライ回数（0 でリトライ無効、合計試行回数は MaxRetries+1）
	RetryBaseBackoff time.Duration // 初回バックオフ（係数 4 で増加: 例 500ms → 2s → 8s）
	RetryMaxBackoff  time.Duration // バックオフ上限（Retry-After 含む）
	RetryJitterRatio float64       // ジッター比率（0.2 なら ±20%）
}

// LoadConfig は環境変数からTwelve Dataの設定を読み込みます。
func LoadConfig() Config {
	return Config{
		TwelveDataAPIKey: os.Getenv("TWELVE_DATA_API_KEY"),
		BaseURL:          os.Getenv("TWELVE_DATA_BASE_URL"),
		Timeout:          10 * time.Second,
		MaxRetries:       3,
		RetryBaseBackoff: 500 * time.Millisecond,
		RetryMaxBackoff:  30 * time.Second,
		RetryJitterRatio: 0.2,
	}
}
