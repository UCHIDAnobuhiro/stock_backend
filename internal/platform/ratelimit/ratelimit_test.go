package ratelimit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
)

// setupEvalMock はAllow()のLuaスクリプト実行（EvalSha）のモック期待値を設定します。
// allowed=1, count=現在のカウント を返すように設定します。
// redismockはCustomMatchの前に引数数をチェックするため、ARGV分のダミー引数（5個）を渡します。
func setupEvalMock(mock redismock.ClientMock, key string, allowed int64, count int64) {
	match := mock.CustomMatch(func(expected, actual []interface{}) error {
		return nil
	})
	match.ExpectEvalSha(rateLimitScript.Hash(), []string{key},
		"_", "_", "_", "_", "_"). // ARGV[1]~[5]のダミー値（CustomMatchにより無視される）
		SetVal([]interface{}{allowed, count})
}

// setupEvalErrorMock はAllow()のLuaスクリプト実行がエラーを返すように設定します。
func setupEvalErrorMock(mock redismock.ClientMock, key string, err error) {
	match := mock.CustomMatch(func(expected, actual []interface{}) error {
		return nil
	})
	match.ExpectEvalSha(rateLimitScript.Hash(), []string{key},
		"_", "_", "_", "_", "_").SetErr(err)
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
		count       int64
		limit       int
		wantAllowed bool
		wantRetry   bool
	}{
		{
			name:        "under limit: request allowed",
			count:       2,
			limit:       5,
			wantAllowed: true,
			wantRetry:   false,
		},
		{
			name:        "at limit: request denied",
			count:       5,
			limit:       5,
			wantAllowed: false,
			wantRetry:   true,
		},
		{
			name:        "over limit: request denied",
			count:       10,
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

			var allowed int64
			if tt.wantAllowed {
				allowed = 1
			}
			setupEvalMock(mock, "test:key", allowed, tt.count)

			limiter := NewLimiter(rdb)
			result := limiter.Allow(context.Background(), "test:key", tt.limit, time.Minute)

			assert.Equal(t, tt.wantAllowed, result.Allowed)
			if tt.wantRetry {
				assert.Equal(t, time.Minute, result.RetryAfter)
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
	setupEvalErrorMock(mock, "test:key", connErr)

	limiter := NewLimiter(rdb)
	result := limiter.Allow(context.Background(), "test:key", 5, time.Minute)

	assert.True(t, result.Allowed, "Redisエラー時はリクエストを許可すべき")
	assert.NoError(t, mock.ExpectationsWereMet())
}
