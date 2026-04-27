package main

import (
	"testing"

	candlesusecase "stock_backend/internal/feature/candles/usecase"
)

// TestShouldFailExit はしきい値判定の境界条件を検証します。
// しきい値ちょうど（FailureRate == threshold）は許容し、超過時のみ true を返すこと。
func TestShouldFailExit(t *testing.T) {
	testCases := []struct {
		name      string
		result    candlesusecase.IngestResult
		threshold float64
		want      bool
	}{
		{
			name:      "全銘柄成功 → exit 0",
			result:    candlesusecase.IngestResult{Total: 10, Succeeded: 10, Failed: 0},
			threshold: 0.2,
			want:      false,
		},
		{
			name:      "失敗率がしきい値ちょうど → exit 0（許容）",
			result:    candlesusecase.IngestResult{Total: 10, Succeeded: 8, Failed: 2},
			threshold: 0.2,
			want:      false,
		},
		{
			name:      "失敗率がしきい値超過 → exit 1",
			result:    candlesusecase.IngestResult{Total: 10, Succeeded: 7, Failed: 3},
			threshold: 0.2,
			want:      true,
		},
		{
			name:      "全銘柄失敗 → exit 1",
			result:    candlesusecase.IngestResult{Total: 5, Succeeded: 0, Failed: 5},
			threshold: 0.2,
			want:      true,
		},
		{
			name:      "Total=0（symbol 空） → exit 0",
			result:    candlesusecase.IngestResult{Total: 0},
			threshold: 0.2,
			want:      false,
		},
		{
			name:      "threshold=0 で 1 件失敗 → exit 1（厳格モード）",
			result:    candlesusecase.IngestResult{Total: 10, Succeeded: 9, Failed: 1},
			threshold: 0,
			want:      true,
		},
		{
			name:      "threshold=1.0 で全件失敗 → exit 0（最寛容）",
			result:    candlesusecase.IngestResult{Total: 5, Succeeded: 0, Failed: 5},
			threshold: 1.0,
			want:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldFailExit(tc.result, tc.threshold); got != tc.want {
				t.Errorf("shouldFailExit(%+v, %v)=%v, want %v", tc.result, tc.threshold, got, tc.want)
			}
		})
	}
}
