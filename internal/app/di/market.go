// Package di はアプリケーションコンポーネントの依存性注入ファクトリを提供します。
package di

import (
	"stock_backend/internal/feature/candles/adapters/twelvedata"
	infrahttp "stock_backend/internal/platform/http"
)

// NewMarket はHTTPクライアント付きの完全に設定されたTwelveDataMarketを生成します。
func NewMarket() *twelvedata.TwelveDataMarket {
	cfg := twelvedata.LoadConfig()
	httpClient := infrahttp.NewHTTPClient(cfg.Timeout)
	return twelvedata.NewTwelveDataMarket(cfg, httpClient)
}
