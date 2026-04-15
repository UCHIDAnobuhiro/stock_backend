// Package db はデータベース接続の初期化、リトライ、マイグレーションを提供します。
package db

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

// Validate は Config の必須項目が設定されているかを検証します。
// InstanceName が設定されている場合は Cloud SQL 接続とみなし、Host/Port は不要です。
// それ以外の場合は TCP 接続として Host/Port を必須とします。
// Password は空でも許容します（ローカル開発で空パスワード運用を想定）。
func (c Config) Validate() error {
	var missing []string
	if c.User == "" {
		missing = append(missing, "DB_USER")
	}
	if c.Name == "" {
		missing = append(missing, "DB_NAME")
	}
	if c.InstanceName == "" {
		if c.Host == "" {
			missing = append(missing, "DB_HOST")
		}
		if c.Port == "" {
			missing = append(missing, "DB_PORT")
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missing)
	}
	return nil
}

// BuildDSN は設定からPostgreSQL DSN文字列を構築します。
// InstanceNameが設定されている場合はCloud SQL Unixソケット接続を作成します。
// それ以外の場合はHostとPortを使用してTCP接続を作成します。
func BuildDSN(cfg Config) string {
	if cfg.InstanceName != "" {
		return fmt.Sprintf("host=/cloudsql/%s user=%s password=%s dbname=%s",
			cfg.InstanceName, cfg.User, cfg.Password, cfg.Name)
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name)
}

// Opener はデータベース接続を開くための関数型です。
type Opener func(dsn string) (*gorm.DB, error)

// DefaultOpener はGORMを使用してPostgreSQLデータベースを開きます。
func DefaultOpener(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel()),
	})
}

// gormLogLevel は環境変数 DB_LOG_LEVEL からGORMのログレベルを返します。
// 未設定または不明な値の場合は Silent を返します。
func gormLogLevel() logger.LogLevel {
	switch os.Getenv("DB_LOG_LEVEL") {
	case "info":
		return logger.Info
	case "warn":
		return logger.Warn
	case "error":
		return logger.Error
	default:
		return logger.Silent
	}
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

// RunMigrations は指定されたモデルのデータベースマイグレーションを実行します。
func RunMigrations(db *gorm.DB, models ...any) error {
	return db.AutoMigrate(models...)
}

// OpenDB は環境設定を使用してデータベース接続を開きます。
// リトライロジックを含み、失敗時はプロセスを終了します（本番環境用）。
func OpenDB() *gorm.DB {
	cfg := LoadConfigFromEnv()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid DB config", "error", err)
		os.Exit(1)
	}
	dsn := BuildDSN(cfg)

	db, err := ConnectWithRetry(dsn, 60*time.Second, DefaultOpener)
	if err != nil {
		slog.Error("DB connect failed", "error", err)
		os.Exit(1)
	}

	return db
}
