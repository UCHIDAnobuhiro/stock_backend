# データベースマイグレーション運用ガイド

このディレクトリは [goose](https://github.com/pressly/goose) によるスキーママイグレーションを管理します。

## ディレクトリ構成

```
db/
├── migrations/                  # goose 管理の SQL マイグレーションファイル
│   └── NNNNN_<name>.sql         # `-- +goose Up` と `-- +goose Down` を含む単一ファイル
└── README.md
```

ファイル名は `NNNNN_<snake_case_name>.sql`（5桁連番）で統一します。

## ローカル環境でのコマンド

すべて `goose` CLI を `go tool` 経由で実行します（`tools.go` 不要、`go.mod` の `tool` ディレクティブで管理）。

接続情報は `goose` の DSN 形式（例: `postgres://appuser:apppass@localhost:5432/app?sslmode=disable`）を `GOOSE_DBSTRING` 環境変数で指定します。

```bash
# 環境変数
export GOOSE_DRIVER=postgres
export GOOSE_DBSTRING="postgres://appuser:apppass@localhost:5432/app?sslmode=disable"
export GOOSE_MIGRATION_DIR=db/migrations

# 適用済みマイグレーション状況の確認
go tool goose status

# 最新まで適用
go tool goose up

# 1つロールバック
go tool goose down

# 新しいマイグレーションファイル作成（NNNNN_<name>.sql）
go tool goose create <snake_case_name> sql
```

## 本番・ステージング環境での適用

`docker/Dockerfile.migrate` でビルドしたコンテナを使い、デプロイの前段ジョブで実行します。
詳細は Phase 3 で `cmd/migrate` を `goose` ベースに書き換えた後にこの節を更新します。

## 新規マイグレーションの作成

1. `go tool goose create <name> sql` で雛形を作成
2. `-- +goose Up` セクションに `CREATE` / `ALTER` 等を記述
3. `-- +goose Down` セクションに**必ず**ロールバック SQL を記述
4. ローカル PostgreSQL で `go tool goose up` → `go tool goose down` → `go tool goose up` を試し、可逆性を確認
5. PR に含める

## sqlc コード生成

クエリ追加・変更時は以下を実行して再生成します。

```bash
go tool sqlc generate
```

各 feature の `internal/feature/<name>/adapters/sqlc/queries.sql` を編集 → 同ディレクトリに型安全コードが生成されます。
