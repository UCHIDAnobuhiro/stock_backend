package twelvedata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"stock_backend/internal/domain/entity"
	"stock_backend/internal/domain/repository"
	"stock_backend/internal/interface/dto"
	"strconv"
	"time"
)

type TwelveDataMarket struct {
	cfg    Config
	client *http.Client
}

func NewTwelveDataMarket(cfg Config, client *http.Client) repository.MarketRepository {
	return &TwelveDataMarket{cfg: cfg, client: client}
}

// GetTimeSeries は Twelve Data API から株価の時系列データを取得し、domain.Candle のスライスとして返します。
func (t *TwelveDataMarket) GetTimeSeries(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	q := url.Values{}
	// クエリの追加
	q.Set("symbol", symbol)
	q.Set("interval", interval)
	q.Set("outputsize", strconv.Itoa(outputsize))
	q.Set("apikey", t.cfg.TwelveDataAPIKey)

	// URLの生成
	u := fmt.Sprintf("%s/time_series?%s", t.cfg.BaseURL, q.Encode())

	// リクエストオブジェクトの作成
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	// リクエスト
	res, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("twelvedata http %d", res.StatusCode)
	}

	// dto
	var body dto.TimeSeriesResponse
	// JSONを構造体にデコード
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.Status == "error" {
		return nil, fmt.Errorf("twelvedata: %s", body.Message)
	}

	candles := make([]entity.Candle, 0, len(body.Values))
	for _, v := range body.Values {

		// 時間
		tm, err := time.Parse("2006-01-02 15:04:05", v.Datetime)
		if err != nil {
			tm, err = time.Parse("2006-01-02", v.Datetime)
			if err != nil {
				return nil, fmt.Errorf("parse time %q: %w", v.Datetime, err)
			}
		}
		//　始値
		o, err := strconv.ParseFloat(v.Open, 64)
		if err != nil {
			return nil, fmt.Errorf("parse open %q: %w", v.Open, err)
		}
		// 高値
		h, err := strconv.ParseFloat(v.High, 64)
		if err != nil {
			return nil, fmt.Errorf("parse high %q: %w", v.High, err)
		}
		// 安値
		l, err := strconv.ParseFloat(v.Low, 64)
		if err != nil {
			return nil, fmt.Errorf("parse low %q: %w", v.Low, err)
		}
		// 現在値
		c, err := strconv.ParseFloat(v.Close, 64)
		if err != nil {
			return nil, fmt.Errorf("parse close %q: %w", v.Close, err)
		}
		//出来高
		vol64, err := strconv.ParseInt(v.Volume, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse volume %q: %w", v.Volume, err)
		}

		// domainに変換
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
