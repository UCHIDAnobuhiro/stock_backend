package db

import (
	"fmt"
	"log"
	"os"
	"stock_backend/internal/domain/entity"
	mysqlrepo "stock_backend/internal/infrastructure/mysql"

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

	db, err := gorm.Open(gmysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// マイグレーション（User, Candle など）
	if err := db.AutoMigrate(
		&entity.User{},
		&mysqlrepo.CandleModel{},
	); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	return db
}
