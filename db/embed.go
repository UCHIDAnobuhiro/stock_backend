// Package db はマイグレーション SQL ファイルを Go バイナリに埋め込むためのパッケージです。
// 実行時のファイル配置に依存せず、cmd/migrate などから embed.FS 経由でマイグレーションを参照します。
package db

import "embed"

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrationsFS は db/migrations/*.sql を含む embed.FS を返します。
// goose.SetBaseFS に渡して使用します。
func MigrationsFS() embed.FS {
	return migrationsFS
}

// MigrationsDir は embed.FS 内のマイグレーションディレクトリパスです。
// goose の dir 引数に渡します。
const MigrationsDir = "migrations"
