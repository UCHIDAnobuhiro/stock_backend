package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/gin-contrib/cors"
	"github.com/joho/godotenv"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"stock_backend/internal/domain/entity"
	"stock_backend/internal/infrastructure"
	"stock_backend/internal/infrastructure/externalapi/twelvedata"
	"stock_backend/internal/infrastructure/http"
	"stock_backend/internal/infrastructure/mysql"
	"stock_backend/internal/interface/handler"
	"stock_backend/internal/usecase"
)

func main() {
	// .envを読み込む
	if err := godotenv.Load(".env"); err != nil {
		log.Println("[INFO] .env not found; using system environment variables")
	}

	// DB初期化（今回はSQLite）
	db, err := gorm.Open(sqlite.Open("./stock.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	dbPath, _ := filepath.Abs("./stock.db")
	log.Println("USING_SQLITE:", dbPath)

	// マイグレーション
	if err := db.AutoMigrate(&entity.User{}); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	cfg := twelvedata.LoadConfig()
	httpClient := http.NewHTTPClient(cfg.Timeout)

	// Repository
	userRepo := mysql.NewUserMySQL(db)
	marketRepo := twelvedata.NewTwelveDataMarket(cfg, httpClient)
	symbolRepo := mysql.NewSymbolRepository(db)

	// Usecase
	authUC := usecase.NewAuthUsecase(userRepo)
	candlesUC := usecase.NewCandlesUsecase(marketRepo)
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
