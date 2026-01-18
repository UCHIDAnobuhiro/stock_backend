// Package entity defines the domain models for the candles feature.
package entity

import "time"

// Candle represents OHLCV (Open, High, Low, Close, Volume) candlestick data
// for a stock symbol at a specific time interval.
type Candle struct {
	Symbol   string    // Stock ticker symbol (e.g., "AAPL", "7203.T")
	Interval string    // Time interval (e.g., "1day", "1week", "1month")
	Time     time.Time // Timestamp for the start of this candle period
	Open     float64   // Opening price
	High     float64   // Highest price during this period
	Low      float64   // Lowest price during this period
	Close    float64   // Closing price
	Volume   int64     // Trading volume
}
