package usecase

import (
	"context"
	"stock_backend/internal/domain/repository"
	"time"
)

// IngestUsecase は外部APIからデータを取得し、データベースに永続化するユースケースを定義します。
type IngestUsecase struct {
	market repository.MarketRepository
	candle repository.CandleRepository
}

// NewIngestUsecase は新しい IngestUsecase を作成します。
func NewIngestUsecase(market repository.MarketRepository, candle repository.CandleRepository) *IngestUsecase {
	return &IngestUsecase{market: market, candle: candle}
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
	rl := NewRateLimiter(8, time.Minute) // 1分に8回まで
	for _, s := range symbols {
		for _, interval := range []string{"1day", "1week", "1month"} {
			rl.WaitIfNeeded()
			if err := iu.ingestOne(ctx, s, interval, 200); err != nil {
				// 1つでもエラーが発生したら処理を中断してエラーを返す
				return err
			}
		}
	}
	return nil
}
