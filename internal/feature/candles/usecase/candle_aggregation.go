package usecase

import (
	"fmt"
	"sort"
	"time"

	"stock_backend/internal/feature/candles/domain/entity"
)

// aggregateWeekly はISO週ごとに日足ローソク足を集計して週足を生成します。
// 入力は任意の順序でよく、出力は時刻昇順で返されます。
func aggregateWeekly(daily []entity.Candle) []entity.Candle {
	return aggregate(daily, weekKey, weekStart)
}

// aggregateMonthly は月ごとに日足ローソク足を集計して月足を生成します。
// 入力は任意の順序でよく、出力は時刻昇順で返されます。
func aggregateMonthly(daily []entity.Candle) []entity.Candle {
	return aggregate(daily, monthKey, monthStart)
}

// trimIncompleteFirstBucket は最古の日足がバケット開始日でない場合、最初の集計バケットを除外します。
// 取得データの先頭が週・月の途中から始まる場合に、不完全なバケットで既存の完全なレコードを
// 上書きすることを防ぎます。isBucketStart は与えられた時刻がバケット（週・月）の開始日かどうかを返します。
func trimIncompleteFirstBucket(result []entity.Candle, daily []entity.Candle, isBucketStart func(time.Time) bool) []entity.Candle {
	if len(result) == 0 || len(daily) == 0 {
		return result
	}
	oldest := daily[0].Time
	for _, c := range daily[1:] {
		if c.Time.Before(oldest) {
			oldest = c.Time
		}
	}
	if !isBucketStart(oldest) {
		return result[1:]
	}
	return result
}

// aggregate は日足スライスを keyFn で定義したバケットに集計します。
// startFn はバケットの代表タイムスタンプ（週月の開始日）を返します。
func aggregate(
	daily []entity.Candle,
	keyFn func(time.Time) string,
	startFn func(time.Time) time.Time,
) []entity.Candle {
	if len(daily) == 0 {
		return nil
	}

	// APIは最新順で返すため時刻昇順にソート（Open=初日, Close=末日 を正しく取るため）
	sorted := make([]entity.Candle, len(daily))
	copy(sorted, daily)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Time.Before(sorted[j].Time)
	})

	type bucket struct {
		open   float64
		high   float64
		low    float64
		close  float64
		volume int64
		time   time.Time
	}

	buckets := map[string]*bucket{}
	keyOrder := []string{} // 出現順（= 時刻順）を保持

	for _, c := range sorted {
		k := keyFn(c.Time)
		b, exists := buckets[k]
		if !exists {
			b = &bucket{
				open:   c.Open,
				high:   c.High,
				low:    c.Low,
				close:  c.Close,
				volume: c.Volume,
				time:   startFn(c.Time),
			}
			buckets[k] = b
			keyOrder = append(keyOrder, k)
		} else {
			if c.High > b.high {
				b.high = c.High
			}
			if c.Low < b.low {
				b.low = c.Low
			}
			b.close = c.Close // 昇順ソート済みなので最後の値が終値
			b.volume += c.Volume
		}
	}

	out := make([]entity.Candle, 0, len(keyOrder))
	for _, k := range keyOrder {
		b := buckets[k]
		out = append(out, entity.Candle{
			// Symbol と Interval は呼び出し元（ingestOne）でセットする
			Time:   b.time,
			Open:   b.open,
			High:   b.high,
			Low:    b.low,
			Close:  b.close,
			Volume: b.volume,
		})
	}
	return out
}

// weekKey は ISO 週番号に基づくバケットキーを返します（例: "2023-W01"）。
func weekKey(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

// weekStart はその日が属する ISO 週の月曜日 00:00:00 UTC を返します。
func weekStart(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7 // 日曜日を ISO 準拠で 7 に補正
	}
	monday := t.AddDate(0, 0, -(wd - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}

// monthKey は年月に基づくバケットキーを返します（例: "2023-01"）。
func monthKey(t time.Time) string {
	return fmt.Sprintf("%04d-%02d", t.Year(), int(t.Month()))
}

// monthStart はその日が属する月の 1 日 00:00:00 UTC を返します。
func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}
