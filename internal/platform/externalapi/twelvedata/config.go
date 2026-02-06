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
}

// LoadConfig は環境変数からTwelve Dataの設定を読み込みます。
func LoadConfig() Config {
	return Config{
		TwelveDataAPIKey: os.Getenv("TWELVE_DATA_API_KEY"),
		BaseURL:          os.Getenv("TWELVE_DATA_BASE_URL"),
		Timeout:          10 * time.Second,
	}
}
