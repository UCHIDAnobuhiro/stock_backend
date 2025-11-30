package main

import (
	"log"
	"os"
	"time"

	redisv9 "github.com/redis/go-redis/v9"

	"stock_backend/internal/app/router"
	authadapters "stock_backend/internal/feature/auth/adapters"
	authhandler "stock_backend/internal/feature/auth/transport/handler"
	authusecase "stock_backend/internal/feature/auth/usecase"
	candlesadapters "stock_backend/internal/feature/candles/adapters"
	candleshandler "stock_backend/internal/feature/candles/transport/handler"
	candlesusecase "stock_backend/internal/feature/candles/usecase"
	symbollistadapters "stock_backend/internal/feature/symbollist/adapters"
	symbollisthandler "stock_backend/internal/feature/symbollist/transport/handler"
	symbollistusecase "stock_backend/internal/feature/symbollist/usecase"
	"stock_backend/internal/platform/cache"
	infradb "stock_backend/internal/platform/db"
	jwtmw "stock_backend/internal/platform/jwt"
	infraredis "stock_backend/internal/platform/redis"
)

func main() {
	// db
	db := infradb.OpenDB()

	// Redis
	var rdb *redisv9.Client
	if tmp, err := infraredis.NewRedisClient(); err != nil {
		log.Println("[WARN] Redis unavailable. Running without cache.")
		rdb = nil
	} else {
		rdb = tmp
		defer func() {
			if err := rdb.Close(); err != nil {
				log.Println("[ERROR] Failed to close Redis client:", err)
			}
		}()
	}

	// Repository
	userRepo := authadapters.NewUserMySQL(db)
	symbolRepo := symbollistadapters.NewSymbolRepository(db)
	candleRepo := candlesadapters.NewCandleRepository(db)

	// Redisキャッシュでラップ
	ttl := cache.TimeUntilNext8AM()
	cachedCandleRepo := cache.NewCachingCandleRepository(rdb, ttl, candleRepo, "candles")

	// JWT Generator
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("[FATAL] JWT_SECRET environment variable is required")
	}
	jwtGen := jwtmw.NewGenerator(jwtSecret, 1*time.Hour)

	// Usecase
	authUC := authusecase.NewAuthUsecase(userRepo, jwtGen)
	symbolUC := symbollistusecase.NewSymbolUsecase(symbolRepo)
	candlesUC := candlesusecase.NewCandlesUsecase(cachedCandleRepo)

	// Handler
	authH := authhandler.NewAuthHandler(authUC)
	symbolH := symbollisthandler.NewSymbolHandler(symbolUC)
	candlesH := candleshandler.NewCandlesHandler(candlesUC)

	// ルータ生成
	router := router.NewRouter(authH, candlesH, symbolH)

	// CORS追加 スマホアプリなのでコメントアウト
	// router.Use(cors.Default())

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
