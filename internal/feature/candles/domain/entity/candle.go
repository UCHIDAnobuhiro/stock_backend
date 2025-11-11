package entity

import "time"

type Candle struct {
	Symbol   string
	Interval string
	Time     time.Time
	Open     float64
	High     float64
	Low      float64
	Close    float64
	Volume   int64
}
