package usecase

import (
	"context"
	"log/slog"
	"stock_backend/internal/feature/candles/domain/entity"
	"stock_backend/internal/shared/ratelimiter"
)

const (
	ingestOutputSize = 200 // 1回のリクエストで取得するデータ件数
)

// ingestIntervals はデータ取得の対象となる時間足のリストです。
var ingestIntervals = []string{"1day", "1week", "1month"}

// MarketRepository は株価データを取得するリポジトリのインターフェイスです。
// 外部 API の実装を抽象化します。
// Following Go convention: interfaces are defined by the consumer (usecase), not the provider (adapters).
type MarketRepository interface {
	GetTimeSeries(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
}

// IngestUsecase は外部APIからデータを取得し、データベースに永続化するユースケースを定義します。
type IngestUsecase struct {
	market      MarketRepository
	candle      CandleRepository
	rateLimiter ratelimiter.RateLimiterInterface
}

// NewIngestUsecase は新しい IngestUsecase を作成します。
func NewIngestUsecase(market MarketRepository, candle CandleRepository, rateLimiter ratelimiter.RateLimiterInterface) *IngestUsecase {
	return &IngestUsecase{market: market, candle: candle, rateLimiter: rateLimiter}
}

// ingestOne は指定された銘柄と時間足の時系列データを外部リポジトリから取得し、
// データベースに一括で挿入（または更新）します。
func (iu *IngestUsecase) ingestOne(ctx context.Context, symbol, interval string, outputsize int) error {
	cs, err := iu.market.GetTimeSeries(ctx, symbol, interval, outputsize)
	if err != nil {
		return err
	}

	// 取得したデータに銘柄コードと時間足を設定
	for i := range cs {
		cs[i].Symbol = symbol
		cs[i].Interval = interval
	}
	return iu.candle.UpsertBatch(ctx, cs)
}

// IngestAll は指定された全銘柄の時系列データを複数の時間足（日足, 週足, 月足）で取得し、
// データベースに永続化します。APIのレートリミットを考慮して、リクエスト間に適切な待機時間を設けます。
func (iu *IngestUsecase) IngestAll(ctx context.Context, symbols []string) error {
	for _, s := range symbols {
		for _, interval := range ingestIntervals {
			iu.rateLimiter.WaitIfNeeded()
			if err := iu.ingestOne(ctx, s, interval, ingestOutputSize); err != nil {
				// 1つの銘柄でエラーが発生しても処理を止めずにログに出力し、次の処理を続ける
				slog.Error("failed to ingest data", "symbol", s, "interval", interval, "error", err)
				continue // 次のintervalまたはsymbolへ
			}
		}
	}
	return nil
}
