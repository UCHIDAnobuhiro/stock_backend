package repository

import (
	"context"
	"stock_backend/internal/domain/entity"
)

// MarketRepository　は株価データを取得するリポジトリのインターフェイスです。
// 外部 API の実装を抽象化します。
type MarketRepository interface {
	GetTimeSeries(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
}
