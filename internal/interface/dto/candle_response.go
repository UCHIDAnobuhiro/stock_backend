package dto

// CandleResponse はロウソク足データのレスポンスDTOです。
type CandleResponse struct {
	Time   string  `json:"time"`   // 日付
	Open   float64 `json:"open"`   // 始値
	High   float64 `json:"high"`   // 高値
	Low    float64 `json:"low"`    // 安値
	Close  float64 `json:"close"`  // 終値
	Volume int64   `json:"volume"` // 出来高
}
