package batch

import (
	"log/slog"
)

const (
	rateLimitPerMinute    = 7   // TwelveData APIのレートリミット（無料枠上限8/分、固定ウィンドウずれ対策で1つ余裕を持たせる）
	defaultMaxFailureRate = 0.2 // *_MAX_FAILURE_RATE のデフォルト値
)

// Run は job_id（コマンド引数）に応じてバッチを実行し、終了コードを返す。
// candles: 株価取り込み、logo: ロゴURL取り込み。
// os.Exit は呼ばず、終了コードを返すのみ（呼び出し側の main で os.Exit する）。
func Run(args []string) int {
	if len(args) < 1 {
		slog.Error("job_id is required", "usage", "batch <candles|logo>")
		return 2
	}
	switch args[0] {
	case "candles":
		return runCandleIngest()
	case "logo":
		return runLogoIngest()
	default:
		slog.Error("unknown job_id", "job_id", args[0], "supported", "candles, logo")
		return 2
	}
}
