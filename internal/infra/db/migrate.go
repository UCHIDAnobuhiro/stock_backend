package db

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/pressly/goose/v3"

	dbmigrations "stock_backend/db"
)

// gooseGlobalMu は goose の package-level state (SetDialect / SetBaseFS) への
// アクセスを直列化します。テストで並列に RunGoose を呼んだ際の data race を防ぎます。
var gooseGlobalMu sync.Mutex

// RunGoose は埋め込みマイグレーションに対して goose コマンドを実行します。
// cmd は goose のサブコマンド名（"up", "down", "status", "version" 等）です。
// args は各サブコマンドへの追加引数（例: "up-to 5"）です。
//
// goose v3 の SetDialect / SetBaseFS / RunContext は package-level state を使うため、
// 並列呼び出しを mutex で直列化します（マイグレーション自体は十分速いので影響は小さい）。
func RunGoose(ctx context.Context, db *sql.DB, cmd string, args ...string) error {
	gooseGlobalMu.Lock()
	defer gooseGlobalMu.Unlock()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	migrationsFS := dbmigrations.MigrationsFS()
	goose.SetBaseFS(migrationsFS)
	if err := goose.RunContext(ctx, cmd, db, dbmigrations.MigrationsDir, args...); err != nil {
		return fmt.Errorf("goose %s: %w", cmd, err)
	}
	return nil
}
