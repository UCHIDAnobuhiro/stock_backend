package cache

import (
	"time"
)

// 次の朝8時までのDurationを返す
func TimeUntilNext8AM() time.Duration {
	now := time.Now()
	loc, _ := time.LoadLocation("Asia/Tokyo")

	// 毎朝8時
	next8am := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, loc)

	// すでに8:00を過ぎていたら翌日8:00
	if now.After(next8am) {
		next8am = next8am.Add(24 * time.Hour)
	}

	return next8am.Sub(now)
}
