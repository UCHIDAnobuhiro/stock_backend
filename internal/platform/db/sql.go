package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver "pgx" の登録
)

// SQLOpener は database/sql のコネクションを開く関数型です。
// テストでモック化するために関数型として定義します。
type SQLOpener func(dsn string) (*sql.DB, error)

// DefaultSQLOpener は pgx/v5/stdlib driver で PostgreSQL に接続する SQLOpener です。
// 接続プールのデフォルト値はそのままで、必要に応じて呼び出し側で SetMaxOpenConns 等を調整します。
func DefaultSQLOpener(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	// sql.Open はコネクション確立を遅延するため、Ping で疎通確認する。
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// ConnectSQLWithRetry はリトライ付きで *sql.DB を取得します。
// timeout 期間中、3秒間隔で再試行します。
func ConnectSQLWithRetry(dsn string, timeout time.Duration, opener SQLOpener) (*sql.DB, error) {
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

// OpenSQL は環境変数から設定を読み取り *sql.DB を返します。
// リトライロジックを含み、失敗時は os.Exit します（本番環境用）。
func OpenSQL() *sql.DB {
	cfg := LoadConfigFromEnv()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid DB config", "error", err)
		os.Exit(1)
	}
	if cfg.InstanceName != "" && (cfg.Host != "" || cfg.Port != "") {
		slog.Warn("DB_HOST and DB_PORT are ignored when INSTANCE_CONNECTION_NAME is set",
			"host", cfg.Host, "port", cfg.Port, "instance", cfg.InstanceName)
	}
	dsn := BuildDSN(cfg)

	db, err := ConnectSQLWithRetry(dsn, 60*time.Second, DefaultSQLOpener)
	if err != nil {
		slog.Error("DB connect failed", "error", err)
		os.Exit(1)
	}
	return db
}
