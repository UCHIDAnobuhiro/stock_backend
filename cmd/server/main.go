package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/joho/godotenv"

	"stock_backend/internal/infrastructure"
	"stock_backend/internal/infrastructure/db"
	"stock_backend/internal/infrastructure/mysql"
	"stock_backend/internal/interface/handler"
	"stock_backend/internal/usecase"
)

func main() {
	// .envを読み込む
	if err := godotenv.Load(".env"); err != nil {
		log.Println("[INFO] .env not found; using system environment variables")
	}

	// db
	db := db.OpenDB()

	// Repository
	userRepo := mysql.NewUserMySQL(db)
	marketRepo := infrastructure.NewMarket()
	symbolRepo := mysql.NewSymbolRepository(db)
	candleRepo := mysql.NewCandleRepository(db)

	// Usecase
	authUC := usecase.NewAuthUsecase(userRepo)
	candlesUC := usecase.NewCandlesUsecase(marketRepo, candleRepo)
	symbolUC := usecase.NewSymbolUsecase(symbolRepo)

	// Handler
	authH := handler.NewAuthHandler(authUC)
	candlesH := handler.NewCandlesHandler(candlesUC)
	symbolH := handler.NewSymbolHandler(symbolUC)

	// ルータ生成
	router := infrastructure.NewRouter(authH, candlesH, symbolH)

	// CORS追加
	router.Use(cors.Default())

	// JWT_SECRETチェック（開発中の注意喚起）
	if os.Getenv("JWT_SECRET") == "" {
		log.Println("[WARN] JWT_SECRET is not set. Set a strong secret in production.")
	}

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
