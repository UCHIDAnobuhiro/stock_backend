package redis

import (
	"context"
	"log/slog"
	"os"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient() (*redis.Client, error) {
	addr := os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT")
	password := os.Getenv("REDIS_PASSWORD")

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	// 接続確認
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Error("Redis connection failed", "address", addr, "error", err)
		return nil, err
	}

	slog.Info("Redis connection successful", "address", addr)
	return rdb, nil
}
