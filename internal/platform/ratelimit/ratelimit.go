// Package ratelimit はRedisベースのスライディングウィンドウレートリミッターを提供します。
package ratelimit

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
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

// Allow は指定されたキーに対してリクエストが許可されるかチェックします。
// keyはレートリミットの識別子（例: "rl:login:ip:192.168.1.1"）、
// limitはウィンドウ内の最大リクエスト数、windowはスライディングウィンドウの時間幅です。
func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) Result {
	if l == nil || l.rdb == nil {
		return Result{Allowed: true}
	}

	now := time.Now()
	nowNano := now.UnixNano()
	windowStart := now.Add(-window).UnixNano()

	// Phase 1: ウィンドウ外の古いエントリを削除し、現在のカウントを取得
	pipe := l.rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", windowStart))
	cardCmd := pipe.ZCard(ctx, key)

	if _, err := pipe.Exec(ctx); err != nil {
		slog.Warn("rate limit check failed, allowing request", "key", key, "error", err)
		return Result{Allowed: true}
	}

	count := int(cardCmd.Val())
	if count >= limit {
		return Result{
			Allowed:    false,
			RetryAfter: window,
		}
	}

	// Phase 2: リクエストが許可された場合のみエントリを追加
	var randBuf [8]byte
	_, _ = rand.Read(randBuf[:])
	member := fmt.Sprintf("%d:%x", nowNano, randBuf[:])

	pipe = l.rdb.Pipeline()
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(nowNano), Member: member})
	pipe.Expire(ctx, key, window)

	if _, err := pipe.Exec(ctx); err != nil {
		slog.Warn("rate limit record failed", "key", key, "error", err)
	}

	return Result{Allowed: true}
}
