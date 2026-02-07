package usecase

import (
	"context"
	"log/slog"
	"stock_backend/internal/feature/candles/domain/entity"
	"stock_backend/internal/shared/ratelimiter"
)

const (
	ingestOutputSize = 200 // 1リクエストあたりの取得データポイント数
)

// ingestIntervals はデータ取り込みに使用する時間間隔のリストを定義します。
var ingestIntervals = []string{"1day", "1week", "1month"}

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

// ingestOne は指定された銘柄とインターバルの時系列データを外部リポジトリから取得し、
// データベースにバッチ挿入（または更新）します。
func (iu *IngestUsecase) ingestOne(ctx context.Context, symbol, interval string, outputsize int) error {
	cs, err := iu.market.GetTimeSeries(ctx, symbol, interval, outputsize)
	if err != nil {
		return err
	}

	// 取得したデータにシンボルとインターバルを設定
	for i := range cs {
		cs[i].Symbol = symbol
		cs[i].Interval = interval
	}
	return iu.candle.UpsertBatch(ctx, cs)
}

// IngestAll はアクティブな全銘柄の時系列データを複数の時間間隔（日次、週次、月次）で取得し、
// データベースに永続化します。
// APIレート制限を遵守し、必要に応じてリクエスト間で待機します。
func (iu *IngestUsecase) IngestAll(ctx context.Context) error {
	symbols, err := iu.symbol.ListActiveCodes(ctx)
	if err != nil {
		return err
	}

	for _, s := range symbols {
		for _, interval := range ingestIntervals {
			iu.rateLimiter.WaitIfNeeded()
			if err := iu.ingestOne(ctx, s, interval, ingestOutputSize); err != nil {
				// 1銘柄のエラーで処理を停止せず、エラーをログに記録して続行
				slog.Error("failed to ingest data", "symbol", s, "interval", interval, "error", err)
				continue // 次のインターバルまたは銘柄に進む
			}
		}
	}
	return nil
}
