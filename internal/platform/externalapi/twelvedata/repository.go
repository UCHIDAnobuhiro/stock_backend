package twelvedata

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"stock_backend/internal/feature/candles/domain/entity"
	"stock_backend/internal/feature/candles/usecase"
	"stock_backend/internal/platform/externalapi/twelvedata/dto"
)

// TwelveDataMarket はTwelve Data外部APIから株価データを取得するMarketRepository実装です。
type TwelveDataMarket struct {
	cfg    Config
	client *http.Client
}

// TwelveDataMarketがMarketRepositoryを実装していることをコンパイル時に検証します。
var _ usecase.MarketRepository = (*TwelveDataMarket)(nil)

// NewTwelveDataMarket は指定された設定とHTTPクライアントでTwelveDataMarketの新しいインスタンスを生成します。
func NewTwelveDataMarket(cfg Config, client *http.Client) *TwelveDataMarket {
	return &TwelveDataMarket{cfg: cfg, client: client}
}

// GetTimeSeries はTwelve Data APIから時系列株価データを取得し、
// domain.Candleのスライスとして返します。
func (t *TwelveDataMarket) GetTimeSeries(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	q := url.Values{}
	// クエリパラメータを追加
	q.Set("symbol", symbol)
	q.Set("interval", interval)
	q.Set("outputsize", strconv.Itoa(outputsize))
	q.Set("apikey", t.cfg.TwelveDataAPIKey)

	// URLを生成
	u := fmt.Sprintf("%s/time_series?%s", t.cfg.BaseURL, q.Encode())

	// リクエストオブジェクトを作成
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	// リクエストを実行
	res, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			slog.Warn("failed to close response body", "error", err)
		}
	}()

	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("twelvedata http %d", res.StatusCode)
	}

	// JSONレスポンスをDTOにデコード
	var body dto.TimeSeriesResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.Status == "error" {
		return nil, fmt.Errorf("twelvedata: %s", body.Message)
	}

	candles := make([]entity.Candle, 0, len(body.Values))
	for _, v := range body.Values {

		// タイムスタンプをパース
		tm, err := time.Parse("2006-01-02 15:04:05", v.Datetime)
		if err != nil {
			tm, err = time.Parse("2006-01-02", v.Datetime)
			if err != nil {
				return nil, fmt.Errorf("parse time %q: %w", v.Datetime, err)
			}
		}
		// 始値をパース
		o, err := strconv.ParseFloat(v.Open, 64)
		if err != nil {
			return nil, fmt.Errorf("parse open %q: %w", v.Open, err)
		}
		// 高値をパース
		h, err := strconv.ParseFloat(v.High, 64)
		if err != nil {
			return nil, fmt.Errorf("parse high %q: %w", v.High, err)
		}
		// 安値をパース
		l, err := strconv.ParseFloat(v.Low, 64)
		if err != nil {
			return nil, fmt.Errorf("parse low %q: %w", v.Low, err)
		}
		// 終値をパース
		c, err := strconv.ParseFloat(v.Close, 64)
		if err != nil {
			return nil, fmt.Errorf("parse close %q: %w", v.Close, err)
		}
		// 出来高をパース
		vol64, err := strconv.ParseInt(v.Volume, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse volume %q: %w", v.Volume, err)
		}

		// ドメインエンティティに変換
		candles = append(candles, entity.Candle{
			Time:   tm,
			Open:   o,
			High:   h,
			Low:    l,
			Close:  c,
			Volume: vol64,
		})
	}
	return candles, nil
}
