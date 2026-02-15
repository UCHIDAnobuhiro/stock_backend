package cache

import (
	"time"
)

// TimeUntilNext8AM は次の午前8時（日本時間）までの期間を返します。
func TimeUntilNext8AM() time.Duration {
	loc, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(loc)

	// 次の午前8時を計算
	next8am := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, loc)

	// 今日の午前8時が既に過ぎている場合は明日の午前8時を使用
	if now.After(next8am) {
		next8am = next8am.Add(24 * time.Hour)
	}

	return next8am.Sub(now)
}
