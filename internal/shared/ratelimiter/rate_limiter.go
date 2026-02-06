// Package ratelimiter はAPI呼び出しなどの操作頻度を制限するレートリミッターを提供します。
package ratelimiter

import (
	"log/slog"
	"time"
)

// RateLimiterInterface はAPI呼び出しなどの操作頻度を制限するインターフェースです。
type RateLimiterInterface interface {
	WaitIfNeeded()
}

// RateLimiter はAPI呼び出しなどの操作頻度を制限します。
type RateLimiter struct {
	limit     int           // インターバルあたりの最大操作回数
	interval  time.Duration // カウンターをリセットする時間間隔
	count     int
	lastReset time.Time
}

// NewRateLimiter は新しいRateLimiterインスタンスを生成します。
func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:     limit,
		interval:  interval,
		lastReset: time.Now(),
	}
}

// WaitIfNeeded はレートリミットに達しているか確認し、必要に応じて待機します。
func (rl *RateLimiter) WaitIfNeeded() {
	now := time.Now()
	// インターバルが経過していればカウンターをリセット
	if now.Sub(rl.lastReset) >= rl.interval {
		rl.count = 0
		rl.lastReset = now
	}

	rl.count++
	if rl.count > rl.limit {
		sleep := rl.interval - now.Sub(rl.lastReset)
		if sleep > 0 {
			slog.Info("rate limit reached, sleeping", "limit", rl.limit, "sleep_duration", sleep)
			time.Sleep(sleep)
		}
		// 待機後にリセット
		rl.count = 1
		rl.lastReset = time.Now()
	}
}
