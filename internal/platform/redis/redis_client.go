// Package redis provides Redis client initialization and configuration.
package redis

import (
	"context"
	"log/slog"
	"os"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates a new Redis client using environment variables for configuration.
// It verifies the connection with a PING command before returning.
// Required environment variables: REDIS_HOST, REDIS_PORT, REDIS_PASSWORD (optional).
func NewRedisClient() (*redis.Client, error) {
	addr := os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT")
	password := os.Getenv("REDIS_PASSWORD")

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	// Verify connection
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Error("Redis connection failed", "address", addr, "error", err)
		return nil, err
	}

	slog.Info("Redis connection successful", "address", addr)
	return rdb, nil
}
