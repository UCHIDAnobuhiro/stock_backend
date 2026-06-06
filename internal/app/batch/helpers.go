package batch

import (
	"log/slog"
	"os"
	"strconv"
)

// failureRater は ingest 系 result が共通で実装する失敗率取得インターフェース。
type failureRater interface {
	FailureRate() float64
}

// shouldFailExit は失敗率しきい値から非ゼロ終了すべきかを判定する。
// しきい値ちょうど（FailureRate == threshold）は許容し、超過時のみ true を返す。
func shouldFailExit(result failureRater, threshold float64) bool {
	return result.FailureRate() > threshold
}

// parseTimeoutHours は env のタイムアウト時間（正の整数）を読み取る。未設定・不正時は def を返す。
func parseTimeoutHours(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// parseMaxFailureRate は env の失敗率しきい値（[0,1]）を読み取る。不正時は警告して def を返す。
func parseMaxFailureRate(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if r, err := strconv.ParseFloat(v, 64); err == nil && r >= 0 && r <= 1 {
			return r
		}
		slog.Warn("invalid max failure rate, using default", "key", key, "value", v, "default", def)
	}
	return def
}
