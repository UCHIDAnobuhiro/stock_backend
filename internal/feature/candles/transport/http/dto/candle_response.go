package dto

// CandleResponse is the response DTO for candlestick data.
type CandleResponse struct {
	Time   string  `json:"time"`   // Date
	Open   float64 `json:"open"`   // Opening price
	High   float64 `json:"high"`   // Highest price
	Low    float64 `json:"low"`    // Lowest price
	Close  float64 `json:"close"`  // Closing price
	Volume int64   `json:"volume"` // Trading volume
}
