// Package di provides dependency injection factories for creating application components.
package di

import (
	"stock_backend/internal/platform/externalapi/twelvedata"
	infrahttp "stock_backend/internal/platform/http"
)

// NewMarket creates a fully configured TwelveDataMarket with HTTP client.
func NewMarket() *twelvedata.TwelveDataMarket {
	cfg := twelvedata.LoadConfig()
	httpClient := infrahttp.NewHTTPClient(cfg.Timeout)
	return twelvedata.NewTwelveDataMarket(cfg, httpClient)
}
