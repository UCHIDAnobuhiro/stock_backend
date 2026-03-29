package usecase

import (
	"context"
	"log/slog"

	"stock_backend/internal/feature/candles/domain/entity"
	"stock_backend/internal/shared/ratelimiter"
)

const (
	ingestOutputSize = 5000 // 1リクエストあたりの取得データポイント数（TwelveData 最大値）
)

// CandleWriteRepository はローソク足データの書き込みレイヤーを抽象化します。
// Goの慣例に従い、インターフェースは利用者（usecase）側で定義します。
type CandleWriteRepository interface {
	// UpsertBatch は（symbol, interval, time）をユニークキーとしてUpsert操作を行います。
	UpsertBatch(ctx context.Context, candles []entity.Candle) error
}

// MarketRepository は株式市場データ取得のリポジトリインターフェースを定義します。
// 外部API実装を抽象化します。
// Goの慣例に従い、インターフェースは利用者（usecase）側で定義します。
type MarketRepository interface {
	GetTimeSeries(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
}

// SymbolRepository はデータ取り込み対象の銘柄コード取得を抽象化します。
// Goの慣例に従い、インターフェースは利用者（usecase）側で定義します。
type SymbolRepository interface {
	ListActiveCodes(ctx context.Context) ([]string, error)
}

// IngestUsecase は外部APIからデータを取得し、データベースに永続化するユースケースを定義します。
type IngestUsecase struct {
	market      MarketRepository
	candle      CandleWriteRepository
	symbol      SymbolRepository
	rateLimiter ratelimiter.RateLimiterInterface
}

// NewIngestUsecase はIngestUsecaseの新しいインスタンスを生成します。
func NewIngestUsecase(market MarketRepository, candle CandleWriteRepository, symbol SymbolRepository, rateLimiter ratelimiter.RateLimiterInterface) *IngestUsecase {
	return &IngestUsecase{market: market, candle: candle, symbol: symbol, rateLimiter: rateLimiter}
}

// ingestOne は指定された銘柄の日足データを外部リポジトリから取得し、
// 週足・月足を集計して3種まとめてデータベースにバッチ挿入（または更新）します。
func (iu *IngestUsecase) ingestOne(ctx context.Context, symbol string, outputsize int) error {
	daily, err := iu.market.GetTimeSeries(ctx, symbol, "1day", outputsize)
	if err != nil {
		return err
	}

	for i := range daily {
		daily[i].Symbol = symbol
		daily[i].Interval = "1day"
	}

	weekly := aggregateWeekly(daily)
	for i := range weekly {
		weekly[i].Symbol = symbol
		weekly[i].Interval = "1week"
	}

	monthly := aggregateMonthly(daily)
	for i := range monthly {
		monthly[i].Symbol = symbol
		monthly[i].Interval = "1month"
	}

	all := make([]entity.Candle, 0, len(daily)+len(weekly)+len(monthly))
	all = append(all, daily...)
	all = append(all, weekly...)
	all = append(all, monthly...)

	return iu.candle.UpsertBatch(ctx, all)
}

// IngestAll はアクティブな全銘柄の時系列データを取得し、
// 日足・週足・月足をデータベースに永続化します。
// APIレート制限を遵守し、必要に応じてリクエスト間で待機します。
func (iu *IngestUsecase) IngestAll(ctx context.Context) error {
	symbols, err := iu.symbol.ListActiveCodes(ctx)
	if err != nil {
		return err
	}

	for _, s := range symbols {
		iu.rateLimiter.WaitIfNeeded()
		if err := iu.ingestOne(ctx, s, ingestOutputSize); err != nil {
			// 1銘柄のエラーで処理を停止せず、エラーをログに記録して続行
			slog.Error("failed to ingest data", "symbol", s, "error", err)
		}
	}
	return nil
}
