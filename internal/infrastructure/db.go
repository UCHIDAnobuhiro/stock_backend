package infrastructure

import (
	"fmt"
	"log"
	"os"
	"stock_backend/internal/domain/entity"
	dbmodel "stock_backend/internal/infrastructure/mysql"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func OpenDB() *gorm.DB {
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	name := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, pass, host, port, name)

	fmt.Println("DSN=", dsn)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// マイグレーション（User, Candle など）
	if err := db.AutoMigrate(
		&entity.User{},
		&dbmodel.CandleModel{},
	); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	return db
}
