package db

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestConnectSQLWithRetry_SuccessOnFirstTry は初回接続成功時にリトライせず DB を返すことを検証します。
func TestConnectSQLWithRetry_SuccessOnFirstTry(t *testing.T) {
	t.Parallel()

	mockDB := &sql.DB{}
	calls := 0
	opener := func(dsn string) (*sql.DB, error) {
		calls++
		return mockDB, nil
	}

	db, err := ConnectSQLWithRetry("test-dsn", 5*time.Second, opener)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if db != mockDB {
		t.Error("expected mock DB to be returned")
	}
	if calls != 1 {
		t.Errorf("expected 1 attempt, got %d", calls)
	}
}

// TestConnectSQLWithRetry_RetriesOnFailure は接続失敗時にリトライして最終的に成功することを検証します。
func TestConnectSQLWithRetry_RetriesOnFailure(t *testing.T) {
	// retry interval が 3 秒固定のため Parallel にしない
	mockDB := &sql.DB{}
	attemptCount := 0
	opener := func(dsn string) (*sql.DB, error) {
		attemptCount++
		if attemptCount < 3 {
			return nil, errors.New("connection refused")
		}
		return mockDB, nil
	}

	db, err := ConnectSQLWithRetry("test-dsn", 10*time.Second, opener)
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

// TestConnectSQLWithRetry_TimeoutAfterRetries はタイムアウト後にエラーが返されることを検証します。
func TestConnectSQLWithRetry_TimeoutAfterRetries(t *testing.T) {
	t.Parallel()

	attemptCount := 0
	opener := func(dsn string) (*sql.DB, error) {
		attemptCount++
		return nil, errors.New("connection refused")
	}

	_, err := ConnectSQLWithRetry("test-dsn", 100*time.Millisecond, opener)
	if err == nil {
		t.Fatal("expected error after timeout, got nil")
	}
	if attemptCount == 0 {
		t.Error("expected at least one connection attempt")
	}
}

// TestConnectSQLWithRetry_ErrorWrapped はタイムアウト時のエラーが最後の opener エラーをラップしていることを検証します。
func TestConnectSQLWithRetry_ErrorWrapped(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("driver-specific failure")
	opener := func(dsn string) (*sql.DB, error) {
		return nil, sentinel
	}

	_, err := ConnectSQLWithRetry("test-dsn", 50*time.Millisecond, opener)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected wrapped sentinel error, got %v", err)
	}
}

func TestOpenSQL_InvalidConfig(t *testing.T) {
	t.Setenv("DB_USER", "")

	_, err := OpenSQL()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid DB config") {
		t.Errorf("expected invalid config error, got %v", err)
	}
}

func TestOpenSQLWithRetry_ConnectionFailure(t *testing.T) {
	t.Setenv("DB_USER", "appuser")
	t.Setenv("DB_PASSWORD", "apppass")
	t.Setenv("DB_NAME", "app")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("INSTANCE_CONNECTION_NAME", "")

	sentinel := errors.New("connection refused")
	calls := 0
	opener := func(dsn string) (*sql.DB, error) {
		calls++
		return nil, sentinel
	}

	_, err := openSQLWithRetry(0, opener)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected wrapped sentinel error, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 attempt, got %d", calls)
	}
}
