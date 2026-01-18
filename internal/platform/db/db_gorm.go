package db

import (
	"fmt"
	"log/slog"
	"os"
	"stock_backend/internal/feature/auth/domain/entity"
	candleadapters "stock_backend/internal/feature/candles/adapters"
	"time"

	gmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Config holds database connection configuration.
type Config struct {
	User         string
	Password     string
	Name         string
	Host         string
	Port         string
	InstanceName string // Cloud SQL instance connection name (optional)
}

// LoadConfigFromEnv loads database configuration from environment variables.
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

// BuildDSN constructs a MySQL DSN string from the configuration.
// If InstanceName is set, it creates a Cloud SQL Unix socket connection.
// Otherwise, it creates a TCP connection using Host and Port.
func BuildDSN(cfg Config) string {
	if cfg.InstanceName != "" {
		return fmt.Sprintf("%s:%s@unix(/cloudsql/%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
			cfg.User, cfg.Password, cfg.InstanceName, cfg.Name)
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
}

// Opener is a function type for opening a database connection.
type Opener func(dsn string) (*gorm.DB, error)

// DefaultOpener opens a MySQL database using GORM.
func DefaultOpener(dsn string) (*gorm.DB, error) {
	return gorm.Open(gmysql.Open(dsn), &gorm.Config{})
}

// ConnectWithRetry attempts to connect to the database with retry logic.
// It retries for the specified timeout duration with a 3-second interval.
// Returns the database connection or an error if all retries fail.
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

// RunMigrations runs database migrations for all registered models.
func RunMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&entity.User{},
		&candleadapters.CandleModel{},
	)
}

// OpenDB opens a database connection using environment configuration.
// It includes retry logic and optional migrations based on RUN_MIGRATIONS env var.
// Exits the process on failure (for production use).
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
