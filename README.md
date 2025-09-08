## todo_backend — Gin + クリーンアーキテクチャ + GORM

Go 言語で構築したシンプルな 株価表示アプリの バックエンドです。Gin（HTTP サーバ）、クリーンアーキテクチャ（domain / usecase / interface / infrastructure）、GORM（ORM）を採用しています。デフォルトでは SQLite を利用しますが、DI により MySQL / PostgreSQL などに容易に差し替え可能です。

---

### 特徴

- RESTful API: GET/POST/PUT/DELETE /todos
- クリーンアーキテクチャ構成 + 依存性注入（Repository インターフェースと実装の分離）
- GORM + AutoMigrate によるスキーマ自動生成
- CORS 設定済み（gin-contrib/cors）

---

### 技術スタック

- Go 1.24+
- Gin (github.com/gin-gonic/gin)
- GORM (gorm.io/gorm) + ドライバ（SQLite デフォルト）
- gin-contrib/cors

---

### ディレクトリ構成（簡略）

```
internal/
    domain/ # エンティティ（純粋なビジネスルール）
    usecase/ # アプリケーション固有のユースケース
    interface/
        handler/ # Gin ハンドラー（HTTP I/O）
        repository/ # Repository インターフェース（契約定義）
    infrastructure/
        mysql/ # Repository 実装（GORM 使用）
main.go # 各層の接続とサーバ起動（Composition Root）
```

---

### 実行方法

1. クローン & 依存関係取得

```
git clone <repo-url>
cd todo_backend
go mod tidy
```

2. 起動（SQLite デフォルト）

```
go run cmd/main.go
```

- stock.db が作成され、domain のスキーマが自動適用されます
- Gin サーバが http://localhost:8080 で起動します

---

### クリーンアーキテクチャと DI

- domain: エンティティ（例: Todo）とビジネスルール
- usecase: ユースケースの流れ（TodoUsecase）
- interface/repository: DB アクセス契約（TodoRepository）
- infrastructure/mysql: 実際の DB 実装（GORM）
- interface/handler: Gin ハンドラー（HTTP リクエストとユースケースを接続）
- main.go: 実装を選択して注入し、サーバを起動

💡 DB を差し替える場合は main.go の接続部分だけを変更すれば OK です。
