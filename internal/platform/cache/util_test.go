package cache

import (
	"testing"
	"time"
)

// TestTimeUntilNext8AM は返される期間が正の値かつ24時間以内であることを検証します。
func TestTimeUntilNext8AM(t *testing.T) {
	t.Parallel()

	duration := TimeUntilNext8AM()

	// Duration should always be positive and less than 24 hours
	if duration <= 0 {
		t.Errorf("expected positive duration, got %v", duration)
	}
	if duration > 24*time.Hour {
		t.Errorf("expected duration less than 24 hours, got %v", duration)
	}
}

// TestTimeUntilNext8AM_ReturnsValidDuration は返される期間が次の午前8時（日本時間）までの正確な時間と一致することを検証します。
func TestTimeUntilNext8AM_ReturnsValidDuration(t *testing.T) {
	t.Parallel()

	duration := TimeUntilNext8AM()

	// Calculate what the next 8 AM should be
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("failed to load Asia/Tokyo timezone: %v", err)
	}
	now := time.Now().In(loc)

	next8am := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, loc)
	if now.After(next8am) {
		next8am = next8am.Add(24 * time.Hour)
	}

	// The calculated time should be approximately the same
	expectedDuration := next8am.Sub(now)
	diff := duration - expectedDuration
	if diff < 0 {
		diff = -diff
	}

	// Allow 1 second tolerance for test execution time
	if diff > time.Second {
		t.Errorf("duration %v differs from expected %v by more than 1 second", duration, expectedDuration)
	}
}

// TestTimeUntilNext8AM_AlwaysPositive は複数回呼び出しても常に正の期間が返されることを検証します。
func TestTimeUntilNext8AM_AlwaysPositive(t *testing.T) {
	t.Parallel()

	// Run multiple times to ensure consistency
	for i := 0; i < 10; i++ {
		duration := TimeUntilNext8AM()
		if duration <= 0 {
			t.Errorf("iteration %d: expected positive duration, got %v", i, duration)
		}
	}
}
