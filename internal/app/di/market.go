// Package di はアプリケーションコンポーネントの依存性注入ファクトリを提供します。
package di

import (
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/candles/twelvedata"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/infra/httpclient"
)

// NewMarket はHTTPクライアント付きの完全に設定されたTwelveDataMarketを生成します。
func NewMarket() *twelvedata.TwelveDataMarket {
	cfg := twelvedata.LoadConfig()
	httpClient := httpclient.New(cfg.Timeout)
	return twelvedata.NewTwelveDataMarket(cfg, httpClient)
}
