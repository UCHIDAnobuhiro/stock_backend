package batch

// failureRater は ingest 系 result が共通で実装する失敗率取得インターフェース。
type failureRater interface {
	FailureRate() float64
}

// shouldFailExit は失敗率しきい値から非ゼロ終了すべきかを判定する。
// しきい値ちょうど（FailureRate == threshold）は許容し、超過時のみ true を返す。
func shouldFailExit(result failureRater, threshold float64) bool {
	return result.FailureRate() > threshold
}
