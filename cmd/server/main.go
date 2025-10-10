package main

import (
	"log"
	"os"

	redisv9 "github.com/redis/go-redis/v9"

	"stock_backend/internal/infrastructure"
	"stock_backend/internal/infrastructure/cache"
	infraDB "stock_backend/internal/infrastructure/db"
	"stock_backend/internal/infrastructure/mysql"
	infraRedis "stock_backend/internal/infrastructure/redis"
	"stock_backend/internal/interface/handler"
	"stock_backend/internal/usecase"
)

func main() {
	// db
	db := infraDB.OpenDB()

	// Redis
	var rdb *redisv9.Client
	if tmp, err := infraRedis.NewRedisClient(); err != nil {
		log.Println("[WARN] Redis unavailable. Running without cache.")
		rdb = nil
	} else {
		rdb = tmp
		defer rdb.Close()
	}

	// Repository
	userRepo := mysql.NewUserMySQL(db)
	symbolRepo := mysql.NewSymbolRepository(db)
	candleRepo := mysql.NewCandleRepository(db)

	// Redisキャッシュでラップ
	ttl := cache.TimeUntilNext8AM()
	cachedCandleRepo := cache.NewCachingCandleRepository(rdb, ttl, candleRepo, "candles")

	// Usecase
	authUC := usecase.NewAuthUsecase(userRepo)
	candlesUC := usecase.NewCandlesUsecase(cachedCandleRepo)
	symbolUC := usecase.NewSymbolUsecase(symbolRepo)

	// Handler
	authH := handler.NewAuthHandler(authUC)
	candlesH := handler.NewCandlesHandler(candlesUC)
	symbolH := handler.NewSymbolHandler(symbolUC)

	// ルータ生成
	router := infrastructure.NewRouter(authH, candlesH, symbolH)

	// CORS追加 スマホアプリなのでコメントアウト
	// router.Use(cors.Default())

	// JWT_SECRETチェック（開発中の注意喚起）
	if os.Getenv("JWT_SECRET") == "" {
		log.Println("[WARN] JWT_SECRET is not set. Set a strong secret in production.")
	}

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
