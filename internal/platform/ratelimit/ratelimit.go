// Package ratelimit はRedisベースのスライディングウィンドウレートリミッターを提供します。
package ratelimit

import (
	"context"
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

	pipe := l.rdb.Pipeline()

	// ウィンドウ外の古いエントリを削除
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", windowStart))

	// 現在のエントリ数を取得（ZADD前）
	cardCmd := pipe.ZCard(ctx, key)

	// 現在のリクエストを追加（スコア=ナノ秒タイムスタンプ、メンバー=ナノ秒文字列で一意性確保）
	member := fmt.Sprintf("%d", nowNano)
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(nowNano), Member: member})

	// キーの有効期限を設定（安全ネット）
	pipe.Expire(ctx, key, window)

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

	return Result{Allowed: true}
}
