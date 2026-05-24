package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"

	dbmigrations "stock_backend/db"
)

// RunGoose は埋め込みマイグレーションに対して goose コマンドを実行します。
// cmd は goose のサブコマンド名（"up", "down", "status", "version" 等）です。
// args は各サブコマンドへの追加引数（例: "up-to 5"）です。
//
// 内部で SetDialect / SetBaseFS を都度設定するため、複数回呼び出しても安全です。
func RunGoose(ctx context.Context, db *sql.DB, cmd string, args ...string) error {
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
