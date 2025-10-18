# 🧠 Stock View API (Go / Gin / Clean Architecture)

## 🧭 概要

**株価データ配信と認証を担当するバックエンド API**  
Go 言語と Gin フレームワークを使用し、フロントエンド（Kotlin / Jetpack Compose）と連携します。  
REST API として、ユーザー認証、株価データ提供、キャッシュ最適化を行います。

## ⚙️ 主な機能

- **ユーザー認証**

  - Email / Password によるログイン
  - JWT 発行（短寿命アクセストークン + リフレッシュトークン設計予定）
  - トークン検証ミドルウェアで認可処理

- **株価データ取得**

  - Twelve Data など外部 API から株価データを取得
  - 日足・週足・月足のローソク足（Candlestick）データを返却
  - 最新データのキャッシュ（Redis）に対応

- **キャッシュ最適化**

  - Redis によるローソク足・銘柄データのキャッシュ
  - TTL 設定と自動更新処理
  - キャッシュミス時は API コール＋ DB 保存

- **DB 永続化**
  - MySQL / Cloud SQL によるデータ永続化
  - GORM による ORM 管理

---

## 🛠️ 技術スタック（Tech Stack）

| 分類          | 技術                                                               |
| ------------- | ------------------------------------------------------------------ |
| 言語          | Go (1.24)                                                          |
| Web Framework | Gin                                                                |
| ORM           | GORM                                                               |
| DB            | MySQL / Cloud SQL                                                  |
| Cache         | Redis                                                              |
| Auth          | JWT / bcrypt                                                       |
| Config        | **.docker.env（ローカル） / Secret Manager（本番） + os.Getenv()** |
| Container     | Docker / Docker Compose                                            |
| Cloud         | Google Cloud Run / Cloud SQL / Secret Manager / Artifact Registry  |
| CI/CD         | GitHub Actions                                                     |

## 📂 ディレクトリ構成（Directory Structure）

```text
.
├── cmd/
│   ├── ingest/                 # データ取得・登録処理（定期実行など）
│   └── server/                 # メインエントリポイント（main.go）
│
├── internal/
│   ├── domain/                 # ドメイン層（エンティティ・リポジトリ定義）
│   │   ├── entity/             # ドメインモデル（例：Candle, Symbol）
│   │   └── repository/         # リポジトリインタフェース
│   │
│   ├── infrastructure/         # インフラ層（外部依存モジュール）
│   │   ├── cache/              # キャッシュ処理（例：Redis）
│   │   ├── db/                 # DB接続初期化
│   │   ├── externalapi/        # 外部APIクライアント（Twelve Dataなど）
│   │   ├── http/               # HTTPクライアント設定
│   │   ├── jwt/                # JWT生成・検証処理
│   │   ├── mysql/              # MySQLリポジトリ実装
│   │   └── redis/              # Redisクライアント実装
│   │
│   ├── interface/              # インターフェース層（I/O境界）
│   │   ├── dto/                # DTO（リクエスト／レスポンス構造体）
│   │   └── handler/            # Ginハンドラ・ルーティング
│   │
│   └── usecase/                # アプリケーション層（ユースケース）
│
├── Dockerfile.ingest           # ingest用Dockerfile（本番）
├── Dockerfile.server           # APIサーバ用Dockerfile（本番）
├── Dockerfile.ingest.dev       # ingest開発用Dockerfile（ローカル）
├── docker-compose.yml          # 共通のDocker構成（サービス定義・ネットワーク設定）
├── docker-compose.dev.yml      # ローカル開発用オーバーライド構成
├── .docker.env                 # ローカル環境変数（gitignore推奨）
├── go.mod
├── go.sum
└── .github/
    └── workflows/              # CI/CD（テスト・ビルド・デプロイ）
```

## 🔒 認証設計（JWT + Refresh Token）

### 現在

- JWT アクセストークンによる認証
- `Authorization: Bearer <token>` ヘッダで検証

### 今後（ハイブリッド認証へ）

- **短寿命 JWT（5–10 分）** + **サーバ管理リフレッシュトークン方式** を導入
- `/auth/refresh` によりアクセストークン自動更新
- `/auth/logout` でデバイス単位の即時失効
- リフレッシュトークンは DB または Redis に保存し、**回転（rotate）管理**

## 💾 データフロー（例：株価取得）

1. バッチ処理（`cmd/ingest`）が外部 API（例：Twelve Data）から株価データを取得
2. 取得したローソク足データを MySQL（または Cloud SQL）に保存
3. フロントエンドが `/api/v1/candles?symbol=AAPL&interval=1day` をリクエスト
4. Handler が `CandlesUsecase` を呼び出し
5. Usecase が Repository 経由で **Redis キャッシュを確認**
   - **キャッシュヒット時**：Redis から即時返却
   - **キャッシュミス時**：MySQL からデータを取得 → Redis にキャッシュ → レスポンス返却
6. 結果を JSON でフロントエンドに返却

## 📚 API 仕様（Endpoints）

### 🩺 Health Check

| Method | Path       | 認証 | 説明                                |
| ------ | ---------- | ---- | ----------------------------------- |
| GET    | `/healthz` | 不要 | サービスの稼働確認（200 OK を返却） |

---

### 🔐 認証系（Auth）

| Method | Path      | 認証 | 説明                                 |
| ------ | --------- | ---- | ------------------------------------ |
| POST   | `/signup` | 不要 | 新規ユーザー登録                     |
| POST   | `/login`  | 不要 | ログイン（JWT アクセストークン発行） |

---

### 💹 株価データ系（Candles / Symbols）

| Method | Path             | 認証 | 説明                                         |
| ------ | ---------------- | ---- | -------------------------------------------- |
| GET    | `/symbols`       | 必須 | 銘柄リストの取得                             |
| GET    | `/candles/:code` | 必須 | 指定コードのローソク足データ取得（例：AAPL） |

### 💡 備考

- `/candles` および `/symbols` は **JWT 認証（`Authorization: Bearer <token>`）** が必須です。
- 今後 `/auth/refresh` や `/auth/logout` を追加予定（リフレッシュトークン運用対応）。

## ☁️ Cloud 構成（Google Cloud）

- **Cloud Run**：Docker イメージをデプロイ
- **Cloud SQL (MySQL)**：アプリデータの永続化
- **Redis（Cloud Memorystore）**：キャッシュ管理
- **Secret Manager**：API キー・DB パスワード・JWT 秘密鍵を安全に管理
- 起動時に `os.Getenv()` + Secret Manager API でロード
- **ローカル開発時は `.docker.env` から読み込み**

## 🧪 CI/CD

- **GitHub Actions** によりプルリク作成時に自動テストを実行
- マージ後、**Cloud Build** で Docker イメージをビルドし **Artifact Registry** に保存
- **Workload Identity Federation** を使用して GitHub から GCP へ安全にデプロイ
- **Cloud Run** に自動デプロイし、Secret Manager 経由で環境変数を注入

## ⚙️ 環境構築（Setup）

### 前提

- Docker / Docker Compose がインストール済み
- Go は不要（すべて Docker で起動）
- `.docker.env` にローカル環境変数を設定

---

### 手順

```bash
# クローン
git clone https://github.com/yourname/stock-view-backend.git
cd stock-view-backend

# 環境変数をコピー
cp example.env.docker .env.docker
```

### 🔑 Twelve Data API キーの取得

本アプリでは [Twelve Data API](https://twelvedata.com/) を使用しています。  
株価データを取得するためには無料の API キーが必要です。

1. Twelve Data の公式サイトでアカウントを作成
2. 「Dashboard > API Keys」からキーを発行
3. .docker.env の TWELVE_DATA_API_KEY にコピーして設定  
   例: `TWELVE_DATA_API_KEY=your_api_key_here`

### ⚠️ Twelve Data 無料プランの制約

- 無料プランでは **1 分間に最大 8 リクエスト** まで

そのため本アプリでは、

- **定期バッチ（ingest）でデータを事前取得**
- **Redis キャッシュ** によりリクエストを最小限に抑制  
  という仕組みを採用しています。

### 🧩 API サーバ起動

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml -p stock up backend-dev
```

### 🧠 バッチ処理起動（株価データ取得）

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml -p stock run --rm --no-deps ingest
```

### 💡 備考

- **API サーバ**：<http://localhost:8080>
- **MySQL**：`localhost:3306`
- **Redis**：`localhost:6379`
- **ログ出力**：docker logs -f stock-backend-dev
- **バッチ** ：ingest コンテナが外部 API から株価を取得し、MySQL に保存
