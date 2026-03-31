# ADR-0003: PostgreSQLの採用

| 項目       | 内容       |
| ---------- | ---------- |
| ステータス | Accepted   |
| 日付       | 2026-03-31 |

---

## コンテキスト

stock_backend の永続化層として当初 MySQL を採用していたが、
以下の課題が顕在化した：

- デプロイ先として選定した **Google Cloud Run + Cloud SQL** との組み合わせにおいて、
  Cloud SQL の PostgreSQL インスタンスの方が Google のマネージドサービスとして
  より一般的に推奨されている
- MySQL 固有のエラーコード（1062: 重複キー）を直接参照する実装が必要となり、
  DB 抽象化が薄れていた（`refactor/use-mysql-error-code` PR で対処）
- GORM の PostgreSQL ドライバーはより活発にメンテナンスされており、
  将来的な機能（JSONB・配列型・全文検索）への拡張性が高い

これらを踏まえ、MySQL から PostgreSQL への移行を決定した（PR #117）。

## 決定

永続化データベースとして **PostgreSQL**（Google Cloud SQL for PostgreSQL）を採用する。
ORM には **GORM**（`gorm.io/driver/postgres`）を使用する。
本番環境では Cloud SQL Auth Proxy 経由の **Unix ソケット接続**、
ローカル開発環境では **TCP 接続**（Docker Compose）を使用する。

## 理由

**Cloud SQL for PostgreSQL との親和性**
Cloud Run + Cloud SQL の組み合わせにおいて、Google の公式ドキュメントや
サンプルは PostgreSQL を前提としたものが多く、Cloud SQL Auth Proxy の
Unix ソケット接続（`host=/cloudsql/<instance>`）も PostgreSQL で確立されている。

**DB 固有コードの排除**
MySQL 時代は重複キーエラー判定に MySQL エラーコード 1062 を直接参照していた。
PostgreSQL の GORM ドライバーでは `errors.Is` / GORM の `ErrDuplicatedKey` を
利用できるため、DB 固有の実装が減り抽象化が向上する。

**将来の拡張性**
PostgreSQL は JSONB・配列型・全文検索・LATERAL JOIN など、
株価データの分析クエリや将来的なフィーチャー拡張に有用な機能を標準で持つ。

**GORM の安定したサポート**
`gorm.io/driver/postgres` は GORM 公式のファーストパーティドライバーであり、
メンテナンスが安定している。

## 代替案

| 代替案              | 不採用の理由                                                                           |
| ------------------- | -------------------------------------------------------------------------------------- |
| MySQL の継続利用    | Cloud Run + Cloud SQL 環境での構成が複雑、DB 固有コードの残存、拡張性の低さ            |
| Cloud Spanner       | グローバル分散 RDBMS だが、このプロジェクトのスケールには過剰。費用も高い               |
| SQLite              | 開発・テスト用途には有用だが、Cloud Run のステートレスコンテナ環境では本番利用に不適    |
| MongoDB 等の NoSQL  | 株価のような時系列・集計クエリは RDB の方が適している。スキーマの明確さも保守性に寄与  |

## 影響

### ポジティブな影響

- Cloud SQL Auth Proxy 経由の Unix ソケット接続で、Cloud Run からの接続が簡潔になる
- DB 固有エラーコードへの依存がなくなり、リポジトリ実装の可搬性が向上する
- JSONB・配列型など PostgreSQL 固有の型を将来的に活用できる
- `gorm.AutoMigrate` が PostgreSQL の型システムを正しく扱うため、スキーマ管理が安定する

### ネガティブな影響・トレードオフ

- MySQL から PostgreSQL への移行コストが発生した（全フィーチャーのリポジトリ・シードスクリプト・Docker 設定の更新）
- ローカル開発環境で MySQL から PostgreSQL に切り替えが必要となり、既存の開発環境を再構築する手間が生じた
- SQLite（テスト用）と PostgreSQL（本番）の方言差異（例: `AUTOINCREMENT` vs `SERIAL`）に注意が必要

## 関連ADR

- [ADR-0001](0001-go-言語の採用.md): Go言語の採用
- [ADR-0002](0002-フィーチャーベースのクリーンアーキテクチャ採用.md): フィーチャーベースのクリーンアーキテクチャ採用
