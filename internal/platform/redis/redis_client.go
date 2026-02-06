// Package redis はRedisクライアントの初期化と設定を提供します。
package redis

import (
	"context"
	"log/slog"
	"os"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient は環境変数を使用して設定された新しいRedisクライアントを作成します。
// 返却前にPINGコマンドで接続を検証します。
// 必要な環境変数: REDIS_HOST, REDIS_PORT, REDIS_PASSWORD（オプション）。
func NewRedisClient() (*redis.Client, error) {
	addr := os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT")
	password := os.Getenv("REDIS_PASSWORD")

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	// 接続を検証
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Error("Redis connection failed", "address", addr, "error", err)
		return nil, err
	}

	slog.Info("Redis connection successful", "address", addr)
	return rdb, nil
}
