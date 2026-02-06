# Candles フィーチャー

## 概要

Candlesフィーチャーは、株式市場のローソク足（OHLCV）データ管理を提供します。REST APIによるリアルタイムデータ取得と、外部マーケットデータプロバイダーからのバッチデータ取り込みの両方を処理します。

### 主な機能

- **ローソク足データ取得**: 銘柄、インターバル、出力サイズによるOHLCVデータのクエリ
- **複数の時間間隔**: 日次、週次、月次のインターバルをサポート
- **バッチデータ取り込み**: レート制限付きのTwelveData APIからの自動データ取得
- **Redisキャッシュ**: 自動キャッシュ無効化を備えた透過的なキャッシュレイヤー
- **Upsert操作**: 複合ユニークキーを使用した効率的なバッチ挿入/更新

## シーケンス図

### ローソク足取得フロー（APIリクエスト）

```mermaid
sequenceDiagram
    participant Client
    participant Handler as CandlesHandler
    participant Usecase as CandlesUsecase
    participant Cache as CachingCandleRepository
    participant Repository as CandleMySQL
    participant Redis
    participant DB as MySQL

    Client->>Handler: GET /candles/:code?interval=1day&outputsize=200
    Handler->>Handler: Parse params (defaults: interval=1day, outputsize=200)
    Handler->>Usecase: GetCandles(symbol, interval, outputsize)
    Usecase->>Usecase: Apply defaults if needed
    Usecase->>Cache: Find(symbol, interval, outputsize)

    alt Redis Available
        Cache->>Redis: GET candles:AAPL:1day:200
        alt Cache HIT
            Redis-->>Cache: JSON data
            Cache->>Cache: Unmarshal JSON
            Cache-->>Usecase: []Candle
        else Cache MISS
            Redis-->>Cache: nil
            Cache->>Repository: Find(symbol, interval, outputsize)
            Repository->>DB: SELECT * FROM candles WHERE symbol=? AND interval=? ORDER BY time DESC LIMIT ?
            DB-->>Repository: Rows
            Repository-->>Cache: []Candle
            Cache->>Redis: SET candles:AAPL:1day:200 (TTL)
            Cache-->>Usecase: []Candle
        end
    else Redis Unavailable
        Cache->>Repository: Find(symbol, interval, outputsize)
        Repository->>DB: SELECT ...
        DB-->>Repository: Rows
        Repository-->>Cache: []Candle
        Cache-->>Usecase: []Candle
    end

    Usecase-->>Handler: []Candle
    Handler->>Handler: Convert to DTOs
    Handler-->>Client: 200 OK<br/>[{time, open, high, low, close, volume}, ...]
```

### バッチ取り込みフロー

```mermaid
sequenceDiagram
    participant Main as cmd/ingest/main.go
    participant Usecase as IngestUsecase
    participant RateLimiter
    participant Market as MarketRepository<br/>(TwelveData API)
    participant Cache as CachingCandleRepository
    participant Repository as CandleMySQL
    participant Redis
    participant DB as MySQL

    Main->>Usecase: IngestAll(symbols)

    loop For each symbol
        loop For each interval (1day, 1week, 1month)
            Usecase->>RateLimiter: WaitIfNeeded()
            RateLimiter-->>Usecase: (waits if rate limit reached)

            Usecase->>Market: GetTimeSeries(symbol, interval, 200)
            Market-->>Usecase: []Candle

            Usecase->>Usecase: Set symbol & interval on candles
            Usecase->>Cache: UpsertBatch(candles)
            Cache->>Repository: UpsertBatch(candles)
            Repository->>DB: INSERT ... ON DUPLICATE KEY UPDATE
            DB-->>Repository: Success
            Repository-->>Cache: nil

            alt Redis Available
                Cache->>Redis: SCAN & DEL candles:AAPL:1day:*
                Note over Cache,Redis: Cache invalidation
            end

            Cache-->>Usecase: nil

            alt Error Occurred
                Usecase->>Usecase: Log error, continue to next
            end
        end
    end

    Usecase-->>Main: nil
```

## API仕様

### GET /candles/:code

指定された銘柄のローソク足データを取得します。JWT認証が必要です。

**パスパラメータ**
| パラメータ | 説明 | 例 |
|-----------|------|-----|
| `code` | 銘柄コード | `7203.T`, `AAPL` |

**クエリパラメータ**
| パラメータ | デフォルト | 説明 |
|-----------|-----------|------|
| `interval` | `1day` | 時間間隔（`1day`, `1week`, `1month`） |
| `outputsize` | `200` | 返却するデータポイント数（最大: 5000） |

**リクエスト例**
```
GET /candles/7203.T?interval=1day&outputsize=100
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**レスポンス**

- **200 OK** - 成功
  ```json
  [
    {
      "time": "2024-01-15",
      "open": 2500.0,
      "high": 2550.0,
      "low": 2480.0,
      "close": 2530.0,
      "volume": 1500000
    },
    {
      "time": "2024-01-14",
      "open": 2480.0,
      "high": 2510.0,
      "low": 2470.0,
      "close": 2500.0,
      "volume": 1200000
    }
  ]
  ```
  注: 結果は時間の降順（新しい順）でソートされます。

- **401 Unauthorized** - JWTトークンが未指定または無効
  ```json
  {
    "error": "authorization header required"
  }
  ```

- **502 Bad Gateway** - データベースまたは上流サービスのエラー
  ```json
  {
    "error": "database connection failed"
  }
  ```

## 依存関係図

```mermaid
graph TB
    subgraph "Transport Layer"
        Handler[CandlesHandler<br/>transport/handler]
        DTO[CandleResponse DTO<br/>transport/http/dto]
    end

    subgraph "Usecase Layer"
        CandlesUC[CandlesUsecase<br/>usecase]
        IngestUC[IngestUsecase<br/>usecase]
        RepoInterface[CandleRepository Interface<br/>usecase/candles_usecase.go]
        MarketInterface[MarketRepository Interface<br/>usecase/ingest_usecase.go]
    end

    subgraph "Domain Layer"
        Entity[Candle Entity<br/>domain/entity]
    end

    subgraph "Adapters Layer"
        RepoImpl[CandleMySQL<br/>adapters]
    end

    subgraph "Platform Layer"
        Cache[CachingCandleRepository<br/>platform/cache]
        TwelveData[TwelveDataRepository<br/>platform/externalapi]
        RateLimiter[RateLimiter<br/>shared/ratelimiter]
    end

    subgraph "External Dependencies"
        DB[(MySQL)]
        Redis[(Redis)]
        API[TwelveData API]
    end

    Handler -->|depends on| CandlesUC
    Handler -->|uses| DTO
    CandlesUC -->|defines| RepoInterface
    CandlesUC -->|uses| Entity
    IngestUC -->|defines| MarketInterface
    IngestUC -->|uses| RepoInterface
    IngestUC -->|uses| RateLimiter
    Cache -.->|implements| RepoInterface
    Cache -->|decorates| RepoImpl
    Cache -->|uses| Redis
    RepoImpl -.->|implements| RepoInterface
    RepoImpl -->|uses| Entity
    RepoImpl -->|accesses| DB
    TwelveData -.->|implements| MarketInterface
    TwelveData -->|calls| API

    style Handler fill:#e1f5ff
    style CandlesUC fill:#fff4e1
    style IngestUC fill:#fff4e1
    style Entity fill:#e8f5e9
    style RepoInterface fill:#fff4e1
    style MarketInterface fill:#fff4e1
    style RepoImpl fill:#f3e5f5
    style Cache fill:#f3e5f5
    style TwelveData fill:#f3e5f5
    style DB fill:#ffebee
    style Redis fill:#ffebee
    style API fill:#ffebee
```

### 依存関係の説明

#### トランスポート層（[transport/handler/candle_handler.go](transport/handler/candle_handler.go)）
- **CandlesHandler**: HTTPリクエストを処理し、CandlesUsecaseを呼び出す
- **CandleResponse DTO**（[transport/http/dto/candle_response.go](transport/http/dto/candle_response.go)）: JSONシリアライゼーション付きのレスポンスデータ構造

#### ユースケース層
- **CandlesUsecase**（[usecase/candles_usecase.go](usecase/candles_usecase.go)）: パラメータバリデーション付きのローソク足データ取得
  - インターバルとoutputsizeのデフォルト値を適用
  - 最大outputsize制限（5000）を適用
  - `CandleRepository`インターフェースを定義（Goの「インターフェースは利用者が定義する」慣例に従う）
- **IngestUsecase**（[usecase/ingest_usecase.go](usecase/ingest_usecase.go)）: 外部APIからのバッチデータ取り込み
  - 銘柄とインターバルをイテレーション
  - RateLimiterによるレート制限を遵守
  - `MarketRepository`インターフェースを定義

#### ドメイン層
- **Candle Entity**（[domain/entity/candle.go](domain/entity/candle.go)）: OHLCVローソク足データモデル
  - `Symbol`: 銘柄コード（例: "AAPL", "7203.T"）
  - `Interval`: 時間間隔（例: "1day", "1week", "1month"）
  - `Time`: ローソク足期間のタイムスタンプ
  - `Open`, `High`, `Low`, `Close`: 価格データ
  - `Volume`: 出来高

#### アダプター層（[adapters/candle_mysql.go](adapters/candle_mysql.go)）
- **CandleMySQL**: CandleRepositoryのMySQL実装（GORMを使用）
  - `Find`: 時間の降順でローソク足を取得
  - `UpsertBatch`: `ON DUPLICATE KEY UPDATE`によるバッチ挿入/更新
  - （symbol, interval, time）の複合ユニークインデックス

#### プラットフォーム層
- **CachingCandleRepository**（[platform/cache/caching_candle_repository.go](../../platform/cache/caching_candle_repository.go)）: Redisキャッシュデコレータ
  - CandleMySQLをラップするデコレータパターンを実装
  - キャッシュキー形式: `candles:{symbol}:{interval}:{outputsize}`
  - UpsertBatch時の自動キャッシュ無効化
  - Redis利用不可時のグレースフルデグレード

### アーキテクチャの特徴

1. **クリーンアーキテクチャ**: ドメイン層がインフラストラクチャから独立
2. **依存性逆転**: UsecaseがCandleRepositoryインターフェースを定義し、adaptersが実装
3. **デコレータパターン**: CachingCandleRepositoryが透過的にキャッシュを追加
4. **インターフェース所有権**: インターフェースは利用される場所で定義（Goのベストプラクティス）
5. **グレースフルデグレード**: Redis利用不可時もシステムは継続して動作

## ディレクトリ構成

```
candles/
├── README.md                          # 本ファイル
├── domain/
│   └── entity/
│       └── candle.go                  # Candleエンティティ（OHLCVデータ）
├── usecase/
│   ├── candles_usecase.go             # クエリロジック + CandleRepositoryインターフェース
│   ├── candles_usecase_test.go        # ユースケーステスト
│   ├── ingest_usecase.go              # バッチ取り込み + MarketRepositoryインターフェース
│   └── ingest_usecase_test.go         # 取り込みテスト
├── adapters/
│   ├── candle_mysql.go                # MySQLリポジトリ実装
│   └── candle_mysql_test.go           # リポジトリテスト
└── transport/
    ├── handler/
    │   ├── candle_handler.go          # HTTPハンドラー
    │   └── candle_handler_test.go     # ハンドラーテスト
    └── http/dto/
        └── candle_response.go         # レスポンスDTO
```

## テスト

Candlesフィーチャーの全テストは、一貫性と保守性のために**テーブル駆動テストパターン**に従います。

### テスト構造とパターン

#### 全テスト共通のパターン

1. **テーブル駆動テスト**: 全テスト関数は構造体フィールドを持つ`tests`スライスを使用:
   - `name`: テストケースの説明（例: `"success: all parameters specified"`, `"error: repository returns error"`）
   - `wantErr`: エラーが期待されるかどうかを示すブール値フラグ
   - テストタイプ固有の追加フィールド

2. **並列実行**: リポジトリテストとハンドラーテストは`t.Parallel()`を使用:
   ```go
   func TestCandleMySQL_Find(t *testing.T) {
       t.Parallel()
       // ...
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               t.Parallel()
               // ...
           })
       }
   }
   ```

3. **ヘルパー関数**: 各テストファイルにヘルパー関数を含む:
   - リポジトリ: `setupTestDB()`, `seedCandle()`
   - ハンドラー: HTTPテスト用に`httptest.NewRecorder()`を使用

#### ユースケーステスト（[usecase/candles_usecase_test.go](usecase/candles_usecase_test.go)）

ビジネスロジックを分離してテストするために**モックリポジトリ**を使用します。

**テストケース構造:**
```go
tests := []struct {
    name               string
    inputSymbol        string
    inputInterval      string
    inputOutputsize    int
    mockFindFunc       func(...) ([]entity.Candle, error)
    expectedCandles    []entity.Candle
    expectedErr        error
    expectedInterval   string  // モックに渡されるべき値
    expectedOutputsize int     // モックに渡されるべき値
}{/* ... */}
```

**主な特徴:**
- カスタマイズ可能な動作を持つモック実装
- パラメータバリデーションテスト（デフォルト値、最大値制限）
- 呼び出し回数の検証

**実行コマンド:**
```bash
go test ./internal/feature/candles/usecase/... -v
```

#### ハンドラーテスト（[transport/handler/candle_handler_test.go](transport/handler/candle_handler_test.go)）

HTTPリクエスト/レスポンス処理をテストするために**モックユースケース**を使用します。

**テストケース構造:**
```go
tests := []struct {
    name           string
    url            string
    mockGetCandles func(...) ([]entity.Candle, error)
    expectedStatus int
    expectedBody   string  // JSON文字列比較
}{/* ... */}
```

**主な特徴:**
- HTTPステータスコードの検証
- `assert.JSONEq`によるJSONレスポンスボディの照合
- クエリパラメータのパース検証
- デフォルト値の処理

**実行コマンド:**
```bash
go test ./internal/feature/candles/transport/handler/... -v
```

#### リポジトリテスト（[adapters/candle_mysql_test.go](adapters/candle_mysql_test.go)）

統合テストに**インメモリSQLiteデータベース**を使用します。

**テストケース構造:**
```go
tests := []struct {
    name         string
    symbol       string
    interval     string
    outputsize   int
    wantErr      bool
    setupFunc    func(t *testing.T, db *gorm.DB)
    validateFunc func(t *testing.T, candles []entity.Candle)
}{/* ... */}
```

**主な特徴:**
- 各テストは新しいインメモリSQLiteデータベースを使用
- `setupFunc`: テスト実行前にテストデータを準備
- `validateFunc`: 成功ケースのカスタムバリデーションロジック
- Upsert動作のテスト（挿入 vs 更新）
- ソート順とLIMIT機能のテスト

**実行コマンド:**
```bash
go test ./internal/feature/candles/adapters/... -v
```

### 全テスト実行

```bash
go test ./internal/feature/candles/... -v -race -cover
```

### テスト出力例

```
=== RUN   TestCandlesUsecase_GetCandles
=== RUN   TestCandlesUsecase_GetCandles/success:_all_parameters_specified
=== RUN   TestCandlesUsecase_GetCandles/success:_default_value_used_when_interval_is_empty
=== RUN   TestCandlesUsecase_GetCandles/success:_default_value_used_when_outputsize_is_0
=== RUN   TestCandlesUsecase_GetCandles/success:_default_value_used_when_outputsize_exceeds_max
=== RUN   TestCandlesUsecase_GetCandles/error:_repository_returns_error
--- PASS: TestCandlesUsecase_GetCandles (0.00s)
    --- PASS: TestCandlesUsecase_GetCandles/success:_all_parameters_specified (0.00s)
    --- PASS: TestCandlesUsecase_GetCandles/success:_default_value_used_when_interval_is_empty (0.00s)
    ...
```

## キャッシュ戦略

### キャッシュ設定

| 設定 | 値 | 説明 |
|------|-----|------|
| キー形式 | `candles:{symbol}:{interval}:{outputsize}` | クエリごとの一意なキー |
| デフォルトTTL | 5分 | コンストラクタで設定可能 |
| 名前空間 | `candles` | 分離のためのキープレフィックス |

### キャッシュ動作

1. **読み取りパス（Find）**
   - Redisのキャッシュデータを確認
   - ヒット時: デシリアライズしたデータを返却
   - ミス時: MySQLにクエリ、結果をキャッシュして返却
   - Redisエラー時: キャッシュをバイパスし、MySQLに直接クエリ

2. **書き込みパス（UpsertBatch）**
   - まずMySQLに書き込み
   - パターンマッチングで関連するキャッシュエントリを無効化
   - パターン: `candles:{symbol}:{interval}:*`

### グレースフルデグレード

キャッシュ層はグレースフルに障害を処理するよう設計されています:
- Redis利用不可時、リクエストはMySQLから直接提供
- キャッシュ書き込みの失敗はログに記録されるがリクエストは失敗しない
- 破損したキャッシュエントリは自動的に削除

## 環境変数

| 変数 | 説明 | 必須 |
|------|------|------|
| `TWELVE_DATA_API_KEY` | TwelveDataマーケットデータのAPIキー | はい（取り込み用） |

**注:** RedisとMySQLの接続設定は、このフィーチャー固有ではなくアプリケーションレベルで設定されます。

## 今後の拡張

- WebSocketによるリアルタイムデータストリーミング
- テクニカル指標の追加（SMA、EMA、RSIなど）
- ヒストリカルデータのバックフィル機能
- カスタムインターバルのサポート（5分、15分、1時間）
- CSV/Excel形式でのデータエクスポート
