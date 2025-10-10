package redis

import (
	"context"
	"log"
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
		log.Printf("Redis接続に失敗しました: %v", err)
		return nil, err
	}

	log.Println("Redis接続に成功しました:", addr)
	return rdb, nil
}
