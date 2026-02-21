# Stock View API (Go / Gin / クリーンアーキテクチャ)

## 概要

**株式データ配信・認証用バックエンドAPI**
GoとGinフレームワークで構築し、フロントエンド（Kotlin / Jetpack Compose）と連携します。
REST APIとして、ユーザー認証・株式データ配信・キャッシュ最適化を提供します。

## 主な機能

- **ユーザー認証**

  - メールアドレス/パスワードによるログイン
  - JWTの発行（短期アクセストークン + リフレッシュトークン実装予定）
  - トークン検証ミドルウェアによる認可

- **株式データ取得**

  - 外部API（Twelve Data）からの株式データ取得
  - 日足・週足・月足のローソク足データを返却
  - Redisによる直近データのキャッシュ

- **キャッシュ最適化**

  - ローソク足データ・シンボルデータのRedisキャッシュ
  - TTL設定と自動リフレッシュ
  - キャッシュミス時: API呼び出し + DB保存

- **ロゴ検出・企業分析**

  - 画像からロゴを検出（Cloud Vision API）
  - 検出した企業の分析サマリーを生成（Gemini API / Vertex AI）

- **データベース永続化**
  - MySQL / Cloud SQLによるデータ永続化
  - GORM ORMによるデータ管理

---

## 技術スタック

| カテゴリ        | 技術                                                                |
| --------------- | ------------------------------------------------------------------- |
| 言語            | Go (1.24.13)                                                         |
| Webフレームワーク | Gin                                                                 |
| ORM             | GORM                                                                |
| DB              | MySQL / Cloud SQL                                                   |
| キャッシュ      | Redis                                                               |
| AI / ML         | Cloud Vision API / Gemini API（Vertex AI）                          |
| 認証            | JWT / bcrypt                                                        |
| API仕様         | OpenAPI 3.0.3 / oapi-codegen（型生成）                              |
| 設定管理        | **.env.docker（ローカル）/ Secret Manager（本番）+ os.Getenv()**    |
| コンテナ        | Docker / Docker Compose                                             |
| クラウド        | Google Cloud Run / Cloud SQL / Secret Manager / Artifact Registry   |
| CI/CD           | GitHub Actions                                                      |

## ディレクトリ構成

```text
.
├── api/
│   ├── openapi.yaml            # OpenAPI 3.0.3 仕様（APIコントラクトの単一ソース）
│   └── oapi-codegen.cfg.yaml   # oapi-codegen設定（型のみ生成）
│
├── cmd/
│   ├── ingest/                 # データ取得・取り込み（バッチジョブ）
│   └── server/                 # メインエントリーポイント（main.go）
│
├── internal/
│   ├── api/                    # OpenAPIから自動生成された型定義
│   │   ├── generate.go         # go:generateディレクティブ
│   │   └── types.gen.go        # 生成コード（手動編集不可）
│   │
│   ├── app/                    # アプリケーション基盤
│   │   ├── di/                 # 依存性注入
│   │   └── router/             # ルーティング設定
│   │
│   ├── feature/                # フィーチャーモジュール（垂直スライス）
│   │   ├── auth/               # 認証機能
│   │   │   ├── domain/         # ドメイン層
│   │   │   │   └── entity/     # エンティティ（User）
│   │   │   ├── usecase/        # ユースケース（リポジトリインターフェース定義、ビジネスロジック）
│   │   │   ├── adapters/       # アダプター（リポジトリ実装）
│   │   │   └── transport/
│   │   │       └── handler/    # HTTPハンドラー
│   │   │
│   │   ├── candles/            # ローソク足データ機能
│   │   │   ├── domain/
│   │   │   │   └── entity/     # エンティティ（Candle）
│   │   │   ├── usecase/        # ユースケース（リポジトリインターフェース定義、取得/保存ロジック）
│   │   │   ├── adapters/       # MySQL実装 / Redisキャッシュデコレータ / TwelveData APIクライアント
│   │   │   └── transport/
│   │   │       └── handler/    # HTTPハンドラー
│   │   │
│   │   ├── logodetection/       # ロゴ検出・企業分析機能
│   │   │   ├── domain/
│   │   │   │   └── entity/     # エンティティ（DetectedLogo, CompanyAnalysis）
│   │   │   ├── usecase/        # ユースケース（リポジトリインターフェース定義）
│   │   │   ├── adapters/       # Cloud Vision API / Gemini APIクライアント
│   │   │   └── transport/
│   │   │       └── handler/    # HTTPハンドラー
│   │   │
│   │   └── symbollist/         # シンボルリスト機能
│   │       ├── domain/
│   │       │   └── entity/     # エンティティ（Symbol）
│   │       ├── usecase/        # ユースケース（リポジトリインターフェース定義）
│   │       ├── adapters/       # リポジトリ実装
│   │       └── transport/
│   │           └── handler/    # HTTPハンドラー
│   │
│   ├── platform/               # インフラストラクチャ層（外部依存）
│   │   ├── cache/              # キャッシュユーティリティ（TimeUntilNext8AM等）
│   │   ├── db/                 # データベース接続初期化
│   │   ├── http/               # HTTPクライアント設定
│   │   │   └── handler/        # ヘルスチェックハンドラー
│   │   ├── jwt/                # JWT生成/検証/ミドルウェア
│   │   └── redis/              # Redisクライアント実装
│   │
│   └── shared/                 # 共有ユーティリティ
│       └── ratelimiter/        # レートリミット
│
├── docker/                     # Docker関連ファイル
│   ├── Dockerfile.ingest       # ingest用Dockerfile（本番）
│   ├── Dockerfile.server       # APIサーバー用Dockerfile（本番）
│   ├── Dockerfile.server.dev   # APIサーバー用Dockerfile（ローカル開発）
│   ├── docker-compose.yml      # Docker共通設定（サービス定義・ネットワーク設定）
│   ├── docker-compose.dev.yml  # ローカル開発用オーバーライド設定
│   ├── example.env             # docker-compose変数展開用テンプレート
│   └── mysql/                  # MySQL初期化スクリプト
│
├── .env.docker                 # ローカル環境変数（.gitignoreに追加推奨）
├── go.mod
├── go.sum
└── .github/
    └── workflows/              # CI/CD（テスト、ビルド、デプロイ）
```

## API仕様（OpenAPI）

API仕様は `api/openapi.yaml`（OpenAPI 3.0.3）で管理しています。
この仕様ファイルから [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) を使って Go の型定義を自動生成しています。

### 仕様の確認（Swagger UI）

開発環境の起動時に Swagger UI も自動で立ち上がります：

```bash
docker compose -f docker/docker-compose.yml -f docker/docker-compose.dev.yml -p stock up backend-dev swagger-ui
```

ブラウザで http://localhost:8081 を開くとAPI仕様を確認できます。

### 型の再生成

OpenAPI仕様（`api/openapi.yaml`）を変更した場合、以下のコマンドで Go の型定義を再生成してください：

```bash
go generate ./internal/api/...
```

生成されるファイル: `internal/api/types.gen.go`（手動編集不可）

## 認証設計（JWT + リフレッシュトークン）

### 現在の実装

- JWTアクセストークンによる認証
- `Authorization: Bearer <token>` ヘッダーによる検証

### 今後の計画（ハイブリッド認証）

- **短期JWT（5〜10分）** + **サーバー管理リフレッシュトークン** 方式の実装
- `/auth/refresh` によるアクセストークンの自動更新
- `/auth/logout` によるデバイス単位の即時無効化
- リフレッシュトークンをDBまたはRedisに保存し、**ローテーション管理**を実施

## データフロー（例: 株価取得）

1. バッチプロセス（`cmd/ingest`）が外部API（Twelve Data）から株式データを取得
2. 取得したローソク足データをMySQL（またはCloud SQL）に保存
3. フロントエンドが `GET /v1/candles/AAPL?interval=1day&outputsize=200` をリクエスト（JWT Bearer トークン付き）
4. ハンドラーが `CandlesUsecase` を呼び出し
5. ユースケースがリポジトリ経由で **Redisキャッシュ** を確認
   - **キャッシュヒット**: Redisから即座に返却
   - **キャッシュミス**: MySQLから取得 → Redisにキャッシュ → レスポンスを返却
6. フロントエンドにJSON形式で結果を返却

## APIエンドポイント

### ヘルスチェック

| メソッド | パス       | 認証   | 説明                                    |
| -------- | ---------- | ------ | --------------------------------------- |
| GET      | `/healthz` | 不要   | サービスのヘルスチェック（200 OKを返却） |

---

### 認証

| メソッド | パス         | 認証   | 説明                                  |
| -------- | ------------ | ------ | ------------------------------------- |
| POST     | `/v1/signup` | 不要   | 新規ユーザー登録                      |
| POST     | `/v1/login`  | 不要   | ログイン（JWTアクセストークンを発行） |

---

### 株式データ（ローソク足 / シンボル）

| メソッド | パス                | 認証   | 説明                                              |
| -------- | ------------------- | ------ | ------------------------------------------------- |
| GET      | `/v1/symbols`       | 必要   | シンボルリストの取得                               |
| GET      | `/v1/candles/:code` | 必要   | 指定コードのローソク足データを取得（例: AAPL）     |

---

### ロゴ検出・企業分析

| メソッド | パス                | 認証   | 説明                                              |
| -------- | ------------------- | ------ | ------------------------------------------------- |
| POST     | `/v1/logo/detect`   | 必要   | 画像からロゴを検出（multipart/form-data）          |
| POST     | `/v1/logo/analyze`  | 必要   | 企業分析サマリーを生成（JSON）                     |

### 補足

- `/v1/candles` と `/v1/symbols` は **JWT認証（`Authorization: Bearer <token>`）** が必要です。
- 今後、リフレッシュトークン対応として `/auth/refresh` と `/auth/logout` を追加予定です。

## クラウドアーキテクチャ（Google Cloud）

- **Cloud Run**: Dockerイメージをデプロイ
- **Cloud SQL（MySQL）**: アプリケーションデータの永続化
- **Redis（Cloud Memorystore）**: キャッシュ管理
- **Secret Manager**: APIキー・DBパスワード・JWTシークレットキーを安全に管理
- 起動時に `os.Getenv()` + Secret Manager APIで読み込み
- **ローカル開発では `.env.docker` から読み込み**

## CI/CD

- **GitHub Actions** がプルリクエスト作成時に自動テストを実行
- マージ後、**Cloud Build** がDockerイメージをビルドし、**Artifact Registry** に保存
- **Workload Identity Federation** を使用してGitHubからGCPへ安全にデプロイ
- **Cloud Run** に自動デプロイし、Secret Manager経由で環境変数を注入

## セットアップ

### 前提条件

- Docker / Docker Compose がインストール済みであること
- Go のインストールは不要（すべてDocker内で実行）
- `.env.docker` にローカル環境変数を設定

---

### 手順

```bash
# リポジトリをクローン
git clone https://github.com/UCHIDAnobuhiro/stock_backend.git
cd stock_backend

# 環境変数ファイルをコピー
cp example.env.docker .env.docker
cp docker/example.env docker/.env
```

### Twelve Data APIキーの取得

このアプリケーションは [Twelve Data API](https://twelvedata.com/) を使用しています。
株式データの取得には無料のAPIキーが必要です。

1. Twelve Dataのウェブサイトでアカウントを作成
2. 「Dashboard > API Keys」からキーを発行
3. `.env.docker` に `TWELVE_DATA_API_KEY` として設定
   例: `TWELVE_DATA_API_KEY=your_api_key_here`

### Twelve Data 無料プランの制限事項

- 無料プランでは **1分あたり最大8リクエスト** まで

この制限に対応するため、本アプリケーションでは以下を実施しています：

- **スケジュールバッチ（ingest）プロセスによるデータの事前取得**
- **Redisキャッシュによるリクエスト数の最小化**

### GCP認証の設定（ロゴ検出・企業分析機能を使用する場合）

ロゴ検出・企業分析機能は Google Cloud の Vision API と Vertex AI（Gemini）を使用します。

1. [Google Cloud CLI](https://cloud.google.com/sdk/docs/install) をインストール
2. ADC（Application Default Credentials）で認証

```bash
gcloud auth application-default login
```

3. `docker/.env` に以下を設定

```env
GOOGLE_APPLICATION_CREDENTIALS=/root/.config/gcloud/application_default_credentials.json
HOST_GOOGLE_ADC_PATH=$HOME/.config/gcloud/application_default_credentials.json
```

4. `.env.docker` に以下を追加

```env
GOOGLE_GENAI_USE_VERTEXAI=true
GOOGLE_CLOUD_PROJECT=<GCPプロジェクトID>
GOOGLE_CLOUD_LOCATION=asia-northeast1
```

### APIサーバーの起動

```bash
docker compose -f docker/docker-compose.yml -f docker/docker-compose.dev.yml -p stock up backend-dev
```

### バッチプロセスの起動（株式データ取り込み）

```bash
docker compose -f docker/docker-compose.yml -f docker/docker-compose.dev.yml -p stock run --rm --no-deps ingest
```

### 補足

- **APIサーバー**: <http://localhost:8080>
- **MySQL**: `localhost:3306`
- **Redis**: `localhost:6379`
- **ログ確認**: `docker logs -f stock-backend-dev`
- **バッチプロセス**: ingestコンテナが外部APIから株価を取得し、MySQLに保存
