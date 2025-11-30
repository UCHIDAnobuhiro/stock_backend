package ratelimiter

import (
	"log/slog"
	"time"
)

// RateLimiterInterface defines an interface for limiting the frequency of operations
// such as API calls.
type RateLimiterInterface interface {
	WaitIfNeeded()
}

// RateLimiter limits the frequency of operations such as API calls.
type RateLimiter struct {
	limit     int           // Maximum number of operations per interval
	interval  time.Duration // Time interval for resetting the counter
	count     int
	lastReset time.Time
}

// NewRateLimiter creates a new RateLimiter instance.
func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:     limit,
		interval:  interval,
		lastReset: time.Now(),
	}
}

// WaitIfNeeded checks if the rate limit has been reached and waits if necessary.
func (rl *RateLimiter) WaitIfNeeded() {
	now := time.Now()
	// Reset counter if interval has elapsed
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
		// Reset after waiting
		rl.count = 1
		rl.lastReset = time.Now()
	}
}
