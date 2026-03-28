# CLAUDE.md

このファイルは、Claude Code（claude.ai/code）がこのリポジトリのコードを扱う際のガイダンスを提供します。

## 開発コマンド

### ローカル開発
```bash
# APIサーバー起動（Airによるホットリロード付き）
docker compose -f docker/docker-compose.yml -f docker/docker-compose.dev.yml -p stock up backend-dev

# バッチデータ取り込み実行（外部APIから株価データを取得）
docker compose -f docker/docker-compose.yml -f docker/docker-compose.dev.yml -p stock run --rm --no-deps ingest

# ログ確認
docker logs -f stock-backend-dev
```

### テスト・リント
```bash
# 全テスト実行（レースコンディション検出・カバレッジ付き）
go test ./... -v -race -cover

# 特定パッケージのテスト実行
go test ./internal/feature/candles/usecase/... -v

# 特定テスト関数の実行
go test ./internal/feature/auth/usecase/... -v -run TestAuthUsecase_Login

# リンター実行（golangci-lint、depguardルール使用）
golangci-lint run --timeout=5m

# 全パッケージのビルド
go build ./...
```

### 環境セットアップ
- `example.env.docker` を `.env.docker` にコピーして設定：
  - `TWELVE_DATA_API_KEY`: https://twelvedata.com/ から取得（無料枠: 8リクエスト/分）
  - `JWT_SECRET`: 本番環境では強力なシークレットを設定
  - DB・Redisの設定はローカル開発用

## アーキテクチャ概要

**Go/Gin REST API** で、**フィーチャーベースのクリーンアーキテクチャ**（垂直スライス）を採用しています。

### ディレクトリ構成

```
api/
├── openapi.yaml          # OpenAPI 3.0.3 仕様（APIコントラクトの単一ソース）
└── oapi-codegen.cfg.yaml # oapi-codegen設定（型のみ生成）

internal/
├── api/              # OpenAPIから自動生成された型定義（types.gen.go）
├── app/
│   ├── di/           # 依存性注入ファクトリ
│   └── router/       # HTTPルーター設定
├── feature/          # フィーチャーモジュール（垂直スライス）
│   ├── auth/
│   ├── candles/
│   └── symbollist/
├── platform/         # インフラストラクチャ層（旧 "infrastructure"）
│   ├── cache/        # キャッシュユーティリティ（TimeUntilNext8AM等）
│   ├── db/           # データベース初期化
│   ├── http/         # HTTPクライアント設定 + ヘルスチェックハンドラー
│   ├── jwt/          # JWT生成・ミドルウェア
│   └── redis/        # Redisクライアントセットアップ
└── shared/           # 共有ユーティリティ（例: レートリミッター）
```

### フィーチャーモジュール構成

各フィーチャーは一貫したレイヤー構造に従います：

```
feature/<name>/
├── README.md         # フィーチャーのドキュメント
├── domain/
│   └── entity/       # ドメインモデル（例: Candle, Symbol, User）
├── usecase/          # アプリケーションロジック（リポジトリインターフェース定義、ビジネスロジック統合）
├── adapters/         # リポジトリ実装（MySQL、キャッシュデコレータ、外部APIクライアント等）
└── transport/
    └── handler/      # HTTPハンドラー（Gin）
```

**注意**: リクエスト/レスポンスの型は `internal/api/types.gen.go`（OpenAPI仕様から自動生成）を使用します。各フィーチャーにDTOは配置しません。

**注意**: Goの慣例に従い、**リポジトリインターフェースはusecaseレイヤー**（利用者側）で定義します。別途domain/repositoryディレクトリには配置しません。これにより、インターフェースは使用される場所で定義されます。

### 依存関係ルール（golangci-lint depguardで強制）

1. **レイヤー分離**: **domain/** と **usecase/** は以下をインポート不可：
   - `adapters/`（リポジトリ実装）
   - `transport/`（HTTPハンドラー）
   - `internal/api`（API型定義 - transport層のみ使用可）
2. **フィーチャー分離**: 各フィーチャーは他のフィーチャーをインポート不可
3. **platform分離**: `platform/` は `feature/` をインポート不可

これにより、ドメインロジックがインフラストラクチャの詳細から独立した状態を保ちます。

### 主要なアーキテクチャパターン

1. **リポジトリパターン**: すべてのデータアクセスは `usecase/` レイヤーで定義されたリポジトリインターフェースを経由します（Goの「インターフェースは利用者が定義する」慣例に従う）
2. **キャッシュ用デコレータパターン**: `feature/candles/adapters/CachingCandleRepository` がベースリポジトリをラップ
   - `CandleRepository`（読み取り）と `CandleWriteRepository`（書き込み）の両インターフェースを実装
   - usecaseコードを変更せずにRedisキャッシュを透過的に追加
   - Redisが利用できない場合はグレースフルデグレード（警告ログを出力し、キャッシュなしで動作）
3. **依存性注入**: `cmd/server/main.go` で手動DI
   - Repositories → Usecases → Handlers のワイヤリングは主に main.go で直接実施
   - `internal/app/di/` には一部のファクトリ関数を配置（例: MarketRepositoryの生成）
4. **2つのエントリーポイント**:
   - `cmd/server/main.go`: REST APIサーバー（ポート8080）
   - `cmd/ingest/main.go`: TwelveData APIから株価データを取得するバッチジョブ

### 外部依存
- TwelveData API（株価データ、8リクエスト/分制限） / MySQL（GORM） / Redis（キャッシュ）
- 詳細なデータフローは各フィーチャーの README.md を参照

### 認証
- JWT認証（`platform/jwt/AuthRequired()`）
- 公開: `/healthz`, `/v1/signup`, `/v1/login` / 保護: その他すべて

### テストに関する注意事項

テスト生成の詳細なルール（テーブル駆動テスト、モック定義、レイヤー別戦略等）は `/test-generate` スキル（`.claude/skills/test-generate/SKILL.md`）を参照してください。

## 新機能の追加

新機能を追加する際は、確立されたパターンに従ってください：

1. **フィーチャーディレクトリを作成**: `internal/feature/<feature-name>/` 配下
2. **ドメイン層を最初に定義**:
   - `domain/entity/` - ドメインモデルを作成（純粋なGo構造体）
3. **usecase層を実装**: `usecase/`
   - ここでリポジトリインターフェースを定義（Goの慣例:「インターフェースは利用者が定義する」）
   - リポジトリを統合するビジネスロジックを実装
4. **adaptersを実装**: `adapters/` - usecaseで定義されたインターフェースを実装するリポジトリ実装（MySQL等）
5. **transport層を追加**:
   - `transport/handler/` - HTTPハンドラー（必要に応じてusecaseインターフェースもここで定義可）
   - リクエスト/レスポンス型は `api/openapi.yaml` に定義し、`go generate ./internal/api/...` で生成
6. **依存関係をワイヤリング**: `cmd/server/main.go` または `cmd/ingest/main.go` にて
7. **ルートを登録**: `internal/app/router/router.go` にて
8. **depguardルールを追加**: `.golangci.yml` に以下を追加：
   - `layer-isolation` ルールに新フィーチャーの `adapters` と `transport` パッケージのdenyエントリ
   - 新フィーチャー用の `<name>-isolation` ルール（他フィーチャーへの依存禁止）
   - 既存フィーチャーの isolation ルールに新フィーチャーのdenyエントリ
   - `platform-isolation` ルールに新フィーチャーのdenyエントリ

**重要**: 依存関係ルールを遵守すること - domain/usecaseレイヤーはadaptersやtransportレイヤーをインポートできません。これはgolangci-lint depguardで強制されています。depguardはワイルドカード非対応のため、新フィーチャー追加時に `.golangci.yml` へ明示的にパッケージパスを追加する必要があります。

## コミット・PR作成の言語ルール

コミットメッセージおよびプルリクエストのタイトル・説明はすべて**日本語**で記述してください。

- コミット前のコードレビューは `/code-check` スキル（`.claude/skills/code-check/SKILL.md`）を参照
- コミットメッセージの詳細ルールは `/commit` スキル（`.claude/skills/commit/SKILL.md`）を参照
- プルリクエスト作成の詳細ルールは `/pull-request` スキル（`.claude/skills/pull-request/SKILL.md`）を参照
