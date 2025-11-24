package db

import (
	"fmt"
	"log"
	"os"
	"stock_backend/internal/feature/auth/domain/entity"
	candleadapters "stock_backend/internal/feature/candles/adapters"
	"time"

	gmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func OpenDB() *gorm.DB {
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")

	instance := os.Getenv("INSTANCE_CONNECTION_NAME")

	var dsn string
	if instance != "" {
		dsn = fmt.Sprintf("%s:%s@unix(/cloudsql/%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
			user, pass, instance, name)
	} else {
		host := os.Getenv("DB_HOST")
		port := os.Getenv("DB_PORT")
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
			user, pass, host, port, name)
	}

	var (
		db  *gorm.DB
		err error
	)

	deadline := time.Now().Add(60 * time.Second)
	for {
		db, err = gorm.Open(gmysql.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			log.Fatalf("DB connect failed after 60s: %v", err)
		}
		log.Printf("DB connect failed, retrying...: %v", err)
		time.Sleep(3 * time.Second)
	}

	if os.Getenv("RUN_MIGRATIONS") == "true" {
		// マイグレーション（User, Candle など）
		if err := db.AutoMigrate(
			&entity.User{},
			&candleadapters.CandleModel{},
		); err != nil {
			log.Fatalf("failed to migrate: %v", err)
		}

	}

	return db
}
