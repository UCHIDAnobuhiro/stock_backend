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
internal/
├── app/
│   ├── di/           # 依存性注入ファクトリ
│   └── router/       # HTTPルーター設定
├── feature/          # フィーチャーモジュール（垂直スライス）
│   ├── auth/
│   ├── candles/
│   └── symbollist/
├── platform/         # インフラストラクチャ層（旧 "infrastructure"）
│   ├── cache/        # Redisキャッシュデコレータ
│   ├── db/           # データベース初期化
│   ├── externalapi/  # 外部APIクライアント（TwelveData）
│   ├── http/         # HTTPクライアント設定
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
├── adapters/         # リポジトリ実装（MySQL等）
└── transport/
    ├── handler/      # HTTPハンドラー（Gin）
    └── http/dto/     # リクエスト/レスポンスDTO
```

**注意**: Goの慣例に従い、**リポジトリインターフェースはusecaseレイヤー**（利用者側）で定義します。別途domain/repositoryディレクトリには配置しません。これにより、インターフェースは使用される場所で定義されます。

### 依存関係ルール（golangci-lint depguardで強制）

**domain/** と **usecase/** レイヤーは以下をインポートしてはなりません：
- `adapters/`（リポジトリ実装）
- `transport/`（HTTPハンドラー、DTO）

これにより、ドメインロジックがインフラストラクチャの詳細から独立した状態を保ちます。

### 主要なアーキテクチャパターン

1. **リポジトリパターン**: すべてのデータアクセスは `usecase/` レイヤーで定義されたリポジトリインターフェースを経由します（Goの「インターフェースは利用者が定義する」慣例に従う）
2. **キャッシュ用デコレータパターン**: `platform/cache/CachingCandleRepository` がベースリポジトリをラップ
   - 同じ `CandleRepository` インターフェースを実装
   - usecaseコードを変更せずにRedisキャッシュを透過的に追加
   - Redisが利用できない場合はグレースフルデグレード（警告ログを出力し、キャッシュなしで動作）
3. **依存性注入**: `cmd/server/main.go` で手動DI
   - Repositories → Usecases → Handlers のワイヤリングは主に main.go で直接実施
   - `internal/app/di/` には一部のファクトリ関数を配置（例: MarketRepositoryの生成）
4. **2つのエントリーポイント**:
   - `cmd/server/main.go`: REST APIサーバー（ポート8080）
   - `cmd/ingest/main.go`: TwelveData APIから株価データを取得するバッチジョブ

### データフロー例

#### 株価リクエストフロー
1. クライアントがJWT認証付きで `/v1/candles/:code?interval=1day&outputsize=200` をリクエスト
2. ルーター（`app/router`）が `jwtmw.AuthRequired()` ミドルウェアでJWTを検証
3. `candles/transport/handler/CandlesHandler.GetCandlesHandler` にルーティング
4. ハンドラーがパラメータをパース（デフォルト: interval=1day, outputsize=200）し、usecaseを呼び出し
5. Usecaseが `CandleRepository.Find(ctx, symbol, interval, outputsize)` を呼び出し
6. `CachingCandleRepository` がキー形式 `candles:{symbol}:{interval}:{outputsize}` でRedisを確認
   - **キャッシュヒット**: RedisからデシリアライズしたJSONを返却
   - **キャッシュミス**: `candlesadapters.CandleRepository`（MySQL）を呼び出し → 結果をTTL付きでキャッシュ → データを返却
7. ハンドラーがドメインエンティティをDTOに変換してJSONを返却

#### バッチ取り込みフロー
1. `cmd/ingest/main.go` が5分のコンテキストタイムアウトで開始
2. `symbollistadapters.SymbolRepository` からアクティブなシンボルを読み込み
3. 各シンボル × 各インターバル（1day, 1week, 1month）について：
   - `RateLimiter.WaitIfNeeded()` が8リクエスト/分の制限を適用
   - `MarketRepository.GetTimeSeries()` 経由でTwelveData APIを呼び出し
   - `CandleRepository.UpsertBatch()` 経由でMySQLにローソク足データをUpsert
   - キャッシュ無効化: `candles:{symbol}:{interval}:*` に一致するRedisキーを削除
4. エラーはログに記録するが処理は停止しない（次のシンボル/インターバルに継続）

### 外部依存・レートリミット

- **TwelveData API**: 株式市場データプロバイダー（無料枠でレート制限: 8リクエスト/分）
  - レートリミットは `shared/ratelimiter` パッケージで処理
  - 取り込みバッチは3インターバル（1day, 1week, 1month）それぞれ200データポイントを取得
  - レートリミッターは制限到達時に自動的にスリープ
- **MySQL/Cloud SQL**: プライマリデータストア（GORM ORM）
- **Redis**: 動的TTLによるキャッシュ層
  - キャッシュTTLは `cache.TimeUntilNext8AM()` を使用して次の日本時間午前8時（市場開場）に設定
  - キャッシュキーにはシンボル、インターバル、outputsizeを含む
  - キャッシュ無効化は `UpsertBatch` 操作時に実行

### 認証

- `Authorization: Bearer <token>` ヘッダーによるJWTベース認証
- ミドルウェア: `platform/jwt/AuthRequired()`
- 公開エンドポイント: `/healthz`, `/v1/signup`, `/v1/login`
- 保護エンドポイント: `/v1/candles/:code`, `/v1/symbols`

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
   - `transport/http/dto/` - リクエスト/レスポンスDTO
6. **依存関係をワイヤリング**: `cmd/server/main.go` または `cmd/ingest/main.go` にて
7. **ルートを登録**: `internal/app/router/router.go` にて
8. **depguardルールを追加**: `.golangci.yml` に新フィーチャーの `adapters` と `transport` パッケージのdenyルールを追加

**重要**: 依存関係ルールを遵守すること - domain/usecaseレイヤーはadaptersやtransportレイヤーをインポートできません。これはgolangci-lint depguardで強制されています。depguardはワイルドカード非対応のため、新フィーチャー追加時に `.golangci.yml` へ明示的にパッケージパスを追加する必要があります。

## コミットメッセージ・PR作成の言語ルール

コミットメッセージおよびプルリクエストのタイトル・説明はすべて**日本語**で記述してください。

- コミットメッセージの詳細ルールは `/commit` スキル（`.claude/skills/commit/SKILL.md`）を参照
- プルリクエスト作成の詳細ルールは `/pull-request` スキル（`.claude/skills/pull-request/SKILL.md`）を参照
