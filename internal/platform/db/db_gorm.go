package db

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	authentity "stock_backend/internal/feature/auth/domain/entity"
	candleadapters "stock_backend/internal/feature/candles/adapters"
	symbolentity "stock_backend/internal/feature/symbollist/domain/entity"

	gmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Config はデータベース接続設定を保持します。
type Config struct {
	User         string
	Password     string
	Name         string
	Host         string
	Port         string
	InstanceName string // Cloud SQLインスタンス接続名（オプション）
}

// LoadConfigFromEnv は環境変数からデータベース設定を読み込みます。
func LoadConfigFromEnv() Config {
	return Config{
		User:         os.Getenv("DB_USER"),
		Password:     os.Getenv("DB_PASSWORD"),
		Name:         os.Getenv("DB_NAME"),
		Host:         os.Getenv("DB_HOST"),
		Port:         os.Getenv("DB_PORT"),
		InstanceName: os.Getenv("INSTANCE_CONNECTION_NAME"),
	}
}

// BuildDSN は設定からMySQL DSN文字列を構築します。
// InstanceNameが設定されている場合はCloud SQL Unixソケット接続を作成します。
// それ以外の場合はHostとPortを使用してTCP接続を作成します。
func BuildDSN(cfg Config) string {
	if cfg.InstanceName != "" {
		return fmt.Sprintf("%s:%s@unix(/cloudsql/%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
			cfg.User, cfg.Password, cfg.InstanceName, cfg.Name)
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
}

// Opener はデータベース接続を開くための関数型です。
type Opener func(dsn string) (*gorm.DB, error)

// DefaultOpener はGORMを使用してMySQLデータベースを開きます。
func DefaultOpener(dsn string) (*gorm.DB, error) {
	return gorm.Open(gmysql.Open(dsn), &gorm.Config{})
}

// ConnectWithRetry はリトライロジック付きでデータベース接続を試みます。
// 指定されたタイムアウト期間中、3秒間隔でリトライします。
// データベース接続を返すか、すべてのリトライが失敗した場合はエラーを返します。
func ConnectWithRetry(dsn string, timeout time.Duration, opener Opener) (*gorm.DB, error) {
	deadline := time.Now().Add(timeout)
	for {
		db, err := opener(dsn)
		if err == nil {
			return db, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("DB connect failed after %v: %w", timeout, err)
		}
		slog.Warn("DB connect failed, retrying", "error", err)
		time.Sleep(3 * time.Second)
	}
}

// RunMigrations はすべての登録済みモデルのデータベースマイグレーションを実行します。
func RunMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&authentity.User{},
		&candleadapters.CandleModel{},
		&symbolentity.Symbol{},
	)
}

// OpenDB は環境設定を使用してデータベース接続を開きます。
// リトライロジックと、RUN_MIGRATIONS環境変数に基づくオプションのマイグレーションを含みます。
// 失敗時はプロセスを終了します（本番環境用）。
func OpenDB() *gorm.DB {
	cfg := LoadConfigFromEnv()
	dsn := BuildDSN(cfg)

	db, err := ConnectWithRetry(dsn, 60*time.Second, DefaultOpener)
	if err != nil {
		slog.Error("DB connect failed", "error", err)
		os.Exit(1)
	}

	if os.Getenv("RUN_MIGRATIONS") == "true" {
		if err := RunMigrations(db); err != nil {
			slog.Error("failed to migrate", "error", err)
			os.Exit(1)
		}
	}

	return db
}
