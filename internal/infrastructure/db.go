package infrastructure

import (
	"log"
	"path/filepath"
	"stock_backend/internal/domain/entity"
	"stock_backend/internal/infrastructure/mysql"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func OpenDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("./stock.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	dbPath, _ := filepath.Abs("./stock.db")
	log.Println("USING_SQLITE:", dbPath)

	// マイグレーション（User, Candle など）
	if err := db.AutoMigrate(
		&entity.User{},
		&mysql.CandleModel{},
	); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	return db
}
