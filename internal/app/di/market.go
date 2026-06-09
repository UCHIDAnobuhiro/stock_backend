// Package di はアプリケーションコンポーネントの依存性注入ファクトリを提供します。
package di

import (
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/candles/twelvedata"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/infra/httpclient"
)

// NewMarket は渡された設定で、HTTPクライアント付きの完全に設定された TwelveDataMarket を生成します。
// 設定の読み込み（環境変数）は internal/app/config に集約されています。
func NewMarket(cfg twelvedata.Config) *twelvedata.TwelveDataMarket {
	httpClient := httpclient.New(cfg.Timeout)
	return twelvedata.NewTwelveDataMarket(cfg, httpClient)
}
