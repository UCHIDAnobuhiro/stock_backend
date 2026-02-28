package ratelimit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// setupCheckMock はAllow() Phase 1（カウント確認）のRedisパイプラインモック期待値を設定します。
func setupCheckMock(mock redismock.ClientMock, key string, cardVal int64) {
	match := mock.CustomMatch(func(expected, actual []interface{}) error {
		return nil // すべての引数を許可
	})
	match.ExpectZRemRangeByScore(key, "-inf", "0").SetVal(0)
	mock.ExpectZCard(key).SetVal(cardVal)
}

// setupRecordMock はAllow() Phase 2（エントリ追加）のRedisパイプラインモック期待値を設定します。
func setupRecordMock(mock redismock.ClientMock, key string, window time.Duration) {
	match := mock.CustomMatch(func(expected, actual []interface{}) error {
		return nil // すべての引数を許可
	})
	match.ExpectZAdd(key, redis.Z{}).SetVal(1)
	mock.ExpectExpire(key, window).SetVal(true)
}

// TestLimiter_Allow_NilRedis はRedisクライアントがnilの場合にリクエストが許可されることを検証します。
func TestLimiter_Allow_NilRedis(t *testing.T) {
	t.Parallel()

	limiter := NewLimiter(nil)
	result := limiter.Allow(context.Background(), "test:key", 5, time.Minute)

	assert.True(t, result.Allowed, "nil Redisの場合はリクエストを許可すべき")
	assert.Zero(t, result.RetryAfter)
}

// TestLimiter_Allow はスライディングウィンドウレートリミットの許可・拒否判定を検証します。
// 制限内、制限到達時、制限超過時の各ケースをテストします。
func TestLimiter_Allow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cardVal     int64
		limit       int
		wantAllowed bool
		wantRetry   bool
	}{
		{
			name:        "under limit: request allowed",
			cardVal:     2,
			limit:       5,
			wantAllowed: true,
			wantRetry:   false,
		},
		{
			name:        "at limit: request denied",
			cardVal:     5,
			limit:       5,
			wantAllowed: false,
			wantRetry:   true,
		},
		{
			name:        "over limit: request denied",
			cardVal:     10,
			limit:       5,
			wantAllowed: false,
			wantRetry:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rdb, mock := redismock.NewClientMock()
			defer func() { _ = rdb.Close() }()

			window := time.Minute
			setupCheckMock(mock, "test:key", tt.cardVal)
			if tt.wantAllowed {
				setupRecordMock(mock, "test:key", window)
			}

			limiter := NewLimiter(rdb)
			result := limiter.Allow(context.Background(), "test:key", tt.limit, window)

			assert.Equal(t, tt.wantAllowed, result.Allowed)
			if tt.wantRetry {
				assert.Equal(t, window, result.RetryAfter)
			} else {
				assert.Zero(t, result.RetryAfter)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestLimiter_Allow_RedisError_GracefulDegradation はRedis接続エラー時にリクエストが許可されることを検証します。
// グレースフルデグレードにより、Redis障害時もサービスが継続動作することを保証します。
func TestLimiter_Allow_RedisError_GracefulDegradation(t *testing.T) {
	t.Parallel()

	rdb, mock := redismock.NewClientMock()
	defer func() { _ = rdb.Close() }()

	connErr := fmt.Errorf("connection refused")
	match := mock.CustomMatch(func(expected, actual []interface{}) error {
		return nil
	})
	match.ExpectZRemRangeByScore("test:key", "-inf", "0").SetErr(connErr)
	mock.ExpectZCard("test:key").SetErr(connErr)

	limiter := NewLimiter(rdb)
	result := limiter.Allow(context.Background(), "test:key", 5, time.Minute)

	assert.True(t, result.Allowed, "Redisエラー時はリクエストを許可すべき")
}
