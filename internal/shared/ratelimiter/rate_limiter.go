package ratelimiter

import (
	"log/slog"
	"time"
)

// RateLimiterInterface は、API呼び出しなどの操作の頻度を制限するインターフェースです。
type RateLimiterInterface interface {
	WaitIfNeeded()
}

// RateLimiterは、API呼び出しなどの操作の頻度を制限します。
type RateLimiter struct {
	limit     int           // 1分あたりの上限
	interval  time.Duration // どの単位でリセットするか
	count     int
	lastReset time.Time
}

// NewRateLimiterは新しいRateLimiterのインスタンスを生成します。
func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:     limit,
		interval:  interval,
		lastReset: time.Now(),
	}
}

// WaitIfNeededはレートリミットの上限に達しているかを確認し、必要であれば待機します。
func (rl *RateLimiter) WaitIfNeeded() {
	now := time.Now()
	// interval を過ぎたらカウントリセット
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
		// リセット
		rl.count = 1
		rl.lastReset = time.Now()
	}
}
