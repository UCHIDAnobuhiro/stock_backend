// Package ratelimit はRedisベースのスライディングウィンドウレートリミッターを提供します。
package ratelimit

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Result はレートリミットチェックの結果を保持します。
type Result struct {
	Allowed    bool
	RetryAfter time.Duration
}

// Limiter はRedisソート済みセットを使用したスライディングウィンドウレートリミッターです。
// rdbがnilの場合、すべてのリクエストを許可します（グレースフルデグレード）。
type Limiter struct {
	rdb *redis.Client
}

// NewLimiter はLimiterの新しいインスタンスを生成します。
func NewLimiter(rdb *redis.Client) *Limiter {
	return &Limiter{rdb: rdb}
}

// rateLimitScript はスライディングウィンドウレートリミットをRedis上で原子的に実行するLuaスクリプトです。
// KEYS[1]: レートリミットキー
// ARGV[1]: ウィンドウ開始タイムスタンプ（ナノ秒）
// ARGV[2]: 最大リクエスト数（limit）
// ARGV[3]: 現在のタイムスタンプ（ナノ秒、ZADDのスコア）
// ARGV[4]: メンバー値（一意性確保用）
// ARGV[5]: TTL秒数
// 戻り値: {allowed (0 or 1), count}
var rateLimitScript = redis.NewScript(`
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', ARGV[1])
local count = redis.call('ZCARD', KEYS[1])
if count < tonumber(ARGV[2]) then
  redis.call('ZADD', KEYS[1], tonumber(ARGV[3]), ARGV[4])
  redis.call('EXPIRE', KEYS[1], tonumber(ARGV[5]))
  return {1, count}
end
return {0, count}
`)

// Allow は指定されたキーに対してリクエストが許可されるかチェックします。
// keyはレートリミットの識別子（例: "rl:login:ip:192.168.1.1"）、
// limitはウィンドウ内の最大リクエスト数、windowはスライディングウィンドウの時間幅です。
// Luaスクリプトにより判定と追加を原子的に実行し、レースコンディションを防止します。
func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) Result {
	if l == nil || l.rdb == nil {
		return Result{Allowed: true}
	}

	now := time.Now()
	nowNano := now.UnixNano()
	windowStart := now.Add(-window).UnixNano()

	var randBuf [8]byte
	_, _ = rand.Read(randBuf[:])
	member := fmt.Sprintf("%d:%x", nowNano, randBuf[:])

	ttlSeconds := int(window.Seconds())
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}

	res, err := rateLimitScript.Run(ctx, l.rdb, []string{key},
		fmt.Sprintf("%d", windowStart),
		limit,
		fmt.Sprintf("%d", nowNano),
		member,
		ttlSeconds,
	).Int64Slice()

	if err != nil {
		slog.Warn("rate limit check failed, allowing request",
			"prefix", keyPrefix(key), "error", err)
		return Result{Allowed: true}
	}

	if res[0] == 1 {
		return Result{Allowed: true}
	}

	return Result{
		Allowed:    false,
		RetryAfter: window,
	}
}

// ScriptHash はテスト用にLuaスクリプトのSHA1ハッシュを返します。
func ScriptHash() string {
	return rateLimitScript.Hash()
}

// keyPrefix はレートリミットキーからPIIを含まないプレフィックス部分を抽出します。
// 例: "rl:login:email:user@example.com" → "rl:login:email"
func keyPrefix(key string) string {
	if idx := strings.LastIndex(key, ":"); idx > 0 {
		return key[:idx]
	}
	return key
}
