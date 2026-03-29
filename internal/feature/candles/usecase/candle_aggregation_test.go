package usecase

import (
	"testing"
	"time"

	"stock_backend/internal/feature/candles/domain/entity"
)

func mustDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestAggregateWeekly(t *testing.T) {
	testCases := []struct {
		name     string
		input    []entity.Candle
		expected []entity.Candle
	}{
		{
			name:     "empty input returns nil",
			input:    []entity.Candle{},
			expected: nil,
		},
		{
			name: "single candle: correct week start and OHLCV",
			// 2023-06-15 は木曜日 → 週開始は 2023-06-12（月曜）
			input: []entity.Candle{
				{Time: mustDate(2023, 6, 15), Open: 100, High: 110, Low: 90, Close: 105, Volume: 1000},
			},
			expected: []entity.Candle{
				{Time: mustDate(2023, 6, 12), Open: 100, High: 110, Low: 90, Close: 105, Volume: 1000},
			},
		},
		{
			name: "Sunday maps to preceding Monday",
			// 2023-06-18 は日曜 → ISO week 開始は 2023-06-12（月曜）
			input: []entity.Candle{
				{Time: mustDate(2023, 6, 18), Open: 200, High: 220, Low: 190, Close: 210, Volume: 500},
			},
			expected: []entity.Candle{
				{Time: mustDate(2023, 6, 12), Open: 200, High: 220, Low: 190, Close: 210, Volume: 500},
			},
		},
		{
			name: "multiple candles in same week aggregate correctly",
			// 2023-06-12（月）〜 2023-06-14（水）は同一週
			input: []entity.Candle{
				{Time: mustDate(2023, 6, 12), Open: 100, High: 110, Low: 90, Close: 105, Volume: 1000},
				{Time: mustDate(2023, 6, 13), Open: 105, High: 115, Low: 95, Close: 112, Volume: 1200},
				{Time: mustDate(2023, 6, 14), Open: 112, High: 108, Low: 88, Close: 100, Volume: 900},
			},
			expected: []entity.Candle{
				{
					Time:   mustDate(2023, 6, 12),
					Open:   100,  // 最初の日
					High:   115,  // 最大
					Low:    88,   // 最小
					Close:  100,  // 最後の日
					Volume: 3100, // 合計
				},
			},
		},
		{
			name: "candles spanning two weeks produce two results in chronological order",
			// 2023-06-09（金）: ISO W23 / 2023-06-12（月）: ISO W24
			input: []entity.Candle{
				{Time: mustDate(2023, 6, 12), Open: 200, High: 210, Low: 195, Close: 205, Volume: 2000},
				{Time: mustDate(2023, 6, 9), Open: 100, High: 115, Low: 95, Close: 110, Volume: 1000},
			},
			expected: []entity.Candle{
				{Time: mustDate(2023, 6, 5), Open: 100, High: 115, Low: 95, Close: 110, Volume: 1000},   // W23 開始
				{Time: mustDate(2023, 6, 12), Open: 200, High: 210, Low: 195, Close: 205, Volume: 2000}, // W24 開始
			},
		},
		{
			name: "newest-first input still produces correct aggregation",
			// 入力が最新順（APIのデフォルト）でも正しく集計される
			input: []entity.Candle{
				{Time: mustDate(2023, 6, 14), Open: 112, High: 108, Low: 88, Close: 100, Volume: 900},
				{Time: mustDate(2023, 6, 13), Open: 105, High: 115, Low: 95, Close: 112, Volume: 1200},
				{Time: mustDate(2023, 6, 12), Open: 100, High: 110, Low: 90, Close: 105, Volume: 1000},
			},
			expected: []entity.Candle{
				{
					Time:   mustDate(2023, 6, 12),
					Open:   100, // 昇順で最初の日（2023-06-12）の Open
					High:   115,
					Low:    88,
					Close:  100, // 昇順で最後の日（2023-06-14）の Close
					Volume: 3100,
				},
			},
		},
		{
			name: "year-boundary ISO week: 2023-01-01 is in ISO week 2022-W52",
			// 2023-01-01（日）と 2022-12-31（土）は ISO 2022-W52
			input: []entity.Candle{
				{Time: mustDate(2023, 1, 1), Open: 100, High: 110, Low: 90, Close: 105, Volume: 500},
				{Time: mustDate(2022, 12, 31), Open: 95, High: 105, Low: 85, Close: 100, Volume: 600},
			},
			expected: []entity.Candle{
				{
					Time:   mustDate(2022, 12, 26), // ISO 2022-W52 の月曜
					Open:   95,
					High:   110,
					Low:    85,
					Close:  105,
					Volume: 1100,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := aggregateWeekly(tc.input)
			assertCandlesEqual(t, got, tc.expected)
		})
	}
}

func TestAggregateMonthly(t *testing.T) {
	testCases := []struct {
		name     string
		input    []entity.Candle
		expected []entity.Candle
	}{
		{
			name:     "empty input returns nil",
			input:    []entity.Candle{},
			expected: nil,
		},
		{
			name: "single candle: correct month start and OHLCV",
			input: []entity.Candle{
				{Time: mustDate(2023, 6, 15), Open: 100, High: 110, Low: 90, Close: 105, Volume: 1000},
			},
			expected: []entity.Candle{
				{Time: mustDate(2023, 6, 1), Open: 100, High: 110, Low: 90, Close: 105, Volume: 1000},
			},
		},
		{
			name: "multiple candles in same month aggregate correctly",
			input: []entity.Candle{
				{Time: mustDate(2023, 6, 1), Open: 100, High: 110, Low: 90, Close: 105, Volume: 1000},
				{Time: mustDate(2023, 6, 15), Open: 105, High: 120, Low: 95, Close: 115, Volume: 2000},
				{Time: mustDate(2023, 6, 30), Open: 115, High: 112, Low: 88, Close: 110, Volume: 1500},
			},
			expected: []entity.Candle{
				{
					Time:   mustDate(2023, 6, 1),
					Open:   100,
					High:   120,
					Low:    88,
					Close:  110,
					Volume: 4500,
				},
			},
		},
		{
			name: "candles spanning two months produce two results in chronological order",
			input: []entity.Candle{
				{Time: mustDate(2023, 7, 1), Open: 200, High: 210, Low: 195, Close: 205, Volume: 2000},
				{Time: mustDate(2023, 6, 30), Open: 100, High: 115, Low: 95, Close: 110, Volume: 1000},
			},
			expected: []entity.Candle{
				{Time: mustDate(2023, 6, 1), Open: 100, High: 115, Low: 95, Close: 110, Volume: 1000},
				{Time: mustDate(2023, 7, 1), Open: 200, High: 210, Low: 195, Close: 205, Volume: 2000},
			},
		},
		{
			name: "year boundary: December and January produce separate monthly candles",
			input: []entity.Candle{
				{Time: mustDate(2023, 1, 1), Open: 100, High: 110, Low: 90, Close: 105, Volume: 500},
				{Time: mustDate(2022, 12, 31), Open: 95, High: 105, Low: 85, Close: 100, Volume: 600},
			},
			expected: []entity.Candle{
				{Time: mustDate(2022, 12, 1), Open: 95, High: 105, Low: 85, Close: 100, Volume: 600},
				{Time: mustDate(2023, 1, 1), Open: 100, High: 110, Low: 90, Close: 105, Volume: 500},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := aggregateMonthly(tc.input)
			assertCandlesEqual(t, got, tc.expected)
		})
	}
}

// assertCandlesEqual は2つの Candle スライスを比較します（Symbol/Interval は無視）。
func assertCandlesEqual(t *testing.T, got, want []entity.Candle) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
		return
	}
	if len(got) != len(want) {
		t.Fatalf("candle count: got %d, want %d\ngot:  %+v\nwant: %+v", len(got), len(want), got, want)
	}
	for i := range want {
		if !got[i].Time.Equal(want[i].Time) {
			t.Errorf("[%d] Time: got %v, want %v", i, got[i].Time, want[i].Time)
		}
		if got[i].Open != want[i].Open {
			t.Errorf("[%d] Open: got %v, want %v", i, got[i].Open, want[i].Open)
		}
		if got[i].High != want[i].High {
			t.Errorf("[%d] High: got %v, want %v", i, got[i].High, want[i].High)
		}
		if got[i].Low != want[i].Low {
			t.Errorf("[%d] Low: got %v, want %v", i, got[i].Low, want[i].Low)
		}
		if got[i].Close != want[i].Close {
			t.Errorf("[%d] Close: got %v, want %v", i, got[i].Close, want[i].Close)
		}
		if got[i].Volume != want[i].Volume {
			t.Errorf("[%d] Volume: got %v, want %v", i, got[i].Volume, want[i].Volume)
		}
	}
}
