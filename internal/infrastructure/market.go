// internal/infrastructure/market.go
package infrastructure

import (
	"stock_backend/internal/domain/repository"
	"stock_backend/internal/infrastructure/externalapi/twelvedata"
	infrahttp "stock_backend/internal/infrastructure/http"
)

func NewMarket() repository.MarketRepository {
	cfg := twelvedata.LoadConfig()
	httpClient := infrahttp.NewHTTPClient(cfg.Timeout)
	return twelvedata.NewTwelveDataMarket(cfg, httpClient)
}
