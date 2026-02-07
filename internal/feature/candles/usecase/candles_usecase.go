// Package usecase はローソク足データ操作のビジネスロジックを実装します。
package usecase

import (
	"context"
	"stock_backend/internal/feature/candles/domain/entity"
)

const (
	// DefaultInterval はローソク足クエリのデフォルト時間間隔です。
	DefaultInterval = "1day"
	// DefaultOutputSize はデフォルトのローソク足返却件数です。
	DefaultOutputSize = 200
	// MaxOutputSize はローソク足の最大返却件数です。
	MaxOutputSize = 5000
)

// CandleRepository はローソク足データの読み取りレイヤーを抽象化します。
// Goの慣例に従い、インターフェースは利用者（usecase）側で定義します。
type CandleRepository interface {
	// Find はデータベースからローソク足データを検索します。
	Find(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
}

// candlesUsecase はローソク足データ操作のユースケースを定義します。
type candlesUsecase struct {
	candle CandleRepository
}

// NewCandlesUsecase はcandlesUsecaseの新しいインスタンスを生成します。
func NewCandlesUsecase(candle CandleRepository) *candlesUsecase {
	return &candlesUsecase{candle: candle}
}

// GetCandles は指定された銘柄と時間間隔のローソク足データを取得します。
func (cu *candlesUsecase) GetCandles(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	if interval == "" {
		interval = DefaultInterval
	}
	if outputsize <= 0 || outputsize > MaxOutputSize {
		outputsize = DefaultOutputSize
	}

	cs, err := cu.candle.Find(ctx, symbol, interval, outputsize)
	if err != nil {
		return nil, err
	}

	return cs, nil
}
