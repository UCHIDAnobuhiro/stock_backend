package db

import (
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"
)

// TestBuildDSN_TCP はTCP接続用のDSN文字列が正しく生成されることを検証します。
func TestBuildDSN_TCP(t *testing.T) {
	t.Parallel()

	cfg := Config{
		User:     "testuser",
		Password: "testpass",
		Name:     "testdb",
		Host:     "localhost",
		Port:     "3306",
	}

	dsn := BuildDSN(cfg)

	expected := "testuser:testpass@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=true&loc=Local"
	if dsn != expected {
		t.Errorf("expected DSN %q, got %q", expected, dsn)
	}
}

// TestBuildDSN_CloudSQL はCloud SQL Unixソケット接続用のDSN文字列が正しく生成されることを検証します。
func TestBuildDSN_CloudSQL(t *testing.T) {
	t.Parallel()

	cfg := Config{
		User:         "testuser",
		Password:     "testpass",
		Name:         "testdb",
		InstanceName: "project:region:instance",
	}

	dsn := BuildDSN(cfg)

	expected := "testuser:testpass@unix(/cloudsql/project:region:instance)/testdb?charset=utf8mb4&parseTime=true&loc=Local"
	if dsn != expected {
		t.Errorf("expected DSN %q, got %q", expected, dsn)
	}
}

// TestBuildDSN_CloudSQLTakesPrecedence はInstanceNameとHost/Portが両方設定されている場合にInstanceNameが優先されることを検証します。
func TestBuildDSN_CloudSQLTakesPrecedence(t *testing.T) {
	t.Parallel()

	// When both InstanceName and Host/Port are set, InstanceName takes precedence
	cfg := Config{
		User:         "testuser",
		Password:     "testpass",
		Name:         "testdb",
		Host:         "localhost",
		Port:         "3306",
		InstanceName: "project:region:instance",
	}

	dsn := BuildDSN(cfg)

	// Should use Cloud SQL format, not TCP
	if dsn == "testuser:testpass@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=true&loc=Local" {
		t.Error("expected Cloud SQL DSN format, but got TCP format")
	}
	expected := "testuser:testpass@unix(/cloudsql/project:region:instance)/testdb?charset=utf8mb4&parseTime=true&loc=Local"
	if dsn != expected {
		t.Errorf("expected DSN %q, got %q", expected, dsn)
	}
}

// TestConnectWithRetry_SuccessOnFirstTry は初回接続成功時にリトライせずDBを返すことを検証します。
func TestConnectWithRetry_SuccessOnFirstTry(t *testing.T) {
	t.Parallel()

	mockDB := &gorm.DB{}
	opener := func(dsn string) (*gorm.DB, error) {
		return mockDB, nil
	}

	db, err := ConnectWithRetry("test-dsn", 5*time.Second, opener)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if db != mockDB {
		t.Error("expected mock DB to be returned")
	}
}

// TestConnectWithRetry_RetriesOnFailure は接続失敗時にリトライして最終的に成功することを検証します。
func TestConnectWithRetry_RetriesOnFailure(t *testing.T) {
	// Not parallel because this test takes time due to retry sleeps

	mockDB := &gorm.DB{}
	attemptCount := 0

	opener := func(dsn string) (*gorm.DB, error) {
		attemptCount++
		if attemptCount < 3 {
			return nil, errors.New("connection refused")
		}
		return mockDB, nil
	}

	// Use a timeout that allows for 2 retries (retry interval is 3 seconds)
	db, err := ConnectWithRetry("test-dsn", 10*time.Second, opener)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if db != mockDB {
		t.Error("expected mock DB to be returned")
	}
	if attemptCount != 3 {
		t.Errorf("expected 3 attempts, got %d", attemptCount)
	}
}

// TestConnectWithRetry_TimeoutAfterRetries はタイムアウト後にエラーが返されることを検証します。
func TestConnectWithRetry_TimeoutAfterRetries(t *testing.T) {
	t.Parallel()

	attemptCount := 0
	opener := func(dsn string) (*gorm.DB, error) {
		attemptCount++
		return nil, errors.New("connection refused")
	}

	// Very short timeout - should fail quickly
	_, err := ConnectWithRetry("test-dsn", 100*time.Millisecond, opener)

	if err == nil {
		t.Fatal("expected error after timeout, got nil")
	}
	if attemptCount == 0 {
		t.Error("expected at least one connection attempt")
	}
}

// TestLoadConfigFromEnv は環境変数からデータベース設定が正しく読み込まれることを検証します。
func TestLoadConfigFromEnv(t *testing.T) {
	// Note: Not running in parallel since we're modifying environment variables
	// Set environment variables for the test
	t.Setenv("DB_USER", "envuser")
	t.Setenv("DB_PASSWORD", "envpass")
	t.Setenv("DB_NAME", "envdb")
	t.Setenv("DB_HOST", "envhost")
	t.Setenv("DB_PORT", "3307")
	t.Setenv("INSTANCE_CONNECTION_NAME", "")

	cfg := LoadConfigFromEnv()

	if cfg.User != "envuser" {
		t.Errorf("expected User 'envuser', got %q", cfg.User)
	}
	if cfg.Password != "envpass" {
		t.Errorf("expected Password 'envpass', got %q", cfg.Password)
	}
	if cfg.Name != "envdb" {
		t.Errorf("expected Name 'envdb', got %q", cfg.Name)
	}
	if cfg.Host != "envhost" {
		t.Errorf("expected Host 'envhost', got %q", cfg.Host)
	}
	if cfg.Port != "3307" {
		t.Errorf("expected Port '3307', got %q", cfg.Port)
	}
}
