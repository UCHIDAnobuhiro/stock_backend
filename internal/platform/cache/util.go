package cache

import (
	"time"
)

// TimeUntilNext8AM returns the duration until the next 8:00 AM (Asia/Tokyo time).
func TimeUntilNext8AM() time.Duration {
	now := time.Now()
	loc, _ := time.LoadLocation("Asia/Tokyo")

	// Calculate next 8:00 AM
	next8am := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, loc)

	// If 8:00 AM has already passed today, use tomorrow's 8:00 AM
	if now.After(next8am) {
		next8am = next8am.Add(24 * time.Hour)
	}

	return next8am.Sub(now)
}
