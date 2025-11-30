package di

import (
	"stock_backend/internal/platform/externalapi/twelvedata"
	infrahttp "stock_backend/internal/platform/http"
)

func NewMarket() *twelvedata.TwelveDataMarket {
	cfg := twelvedata.LoadConfig()
	httpClient := infrahttp.NewHTTPClient(cfg.Timeout)
	return twelvedata.NewTwelveDataMarket(cfg, httpClient)
}
