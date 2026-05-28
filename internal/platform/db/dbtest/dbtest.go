// Package dbtest はリポジトリ層のテストで実 PostgreSQL を使うためのヘルパーを提供します。
//
// 利用パターン:
//
//	func TestMain(m *testing.M) {
//	    code, err := dbtest.RunMainWithPostgres(m)
//	    if err != nil { log.Fatal(err) }
//	    os.Exit(code)
//	}
//
//	func TestXxx(t *testing.T) {
//	    db := dbtest.OpenIsolatedDB(t)
//	    ...
//	}
//
// CI など Docker が利用できる環境では testcontainers-go で PostgreSQL を自動起動します。
// 既存の PostgreSQL を使いたい場合は環境変数 TEST_DB_DSN を設定すると
// コンテナ起動をスキップしてその DSN を「テンプレート DB の管理用接続」として使います。
//
// 各テストは t.Parallel 安全になるよう CREATE DATABASE で独立した DB を持ち、
// テスト終了時に DROP DATABASE します。
package dbtest

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	infradb "stock_backend/internal/platform/db"
)

const envTestDBDSN = "TEST_DB_DSN"

var (
	mu        sync.Mutex
	adminDSN  string // CREATE/DROP DATABASE を発行する管理用 DSN（接続先 DB は問わない）
	dbCounter atomic.Uint64
	container testcontainers.Container
)

// RunMainWithPostgres はテスト用 PostgreSQL を起動して m.Run() を実行し、
// 終了時にコンテナを破棄します。TestMain から呼び出してください。
//
// TEST_DB_DSN が設定されている場合はそれを使用し、コンテナを起動しません。
func RunMainWithPostgres(m *testing.M) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := setup(ctx); err != nil {
		return 1, err
	}
	defer teardown()
	return m.Run(), nil
}

// OpenIsolatedDB はテストごとに独立した PostgreSQL データベースを作成し、
// マイグレーションを適用した *sql.DB を返します。
// t.Cleanup で DB は自動的に DROP されます。
func OpenIsolatedDB(t *testing.T) *sql.DB {
	t.Helper()
	mu.Lock()
	dsn := adminDSN
	mu.Unlock()
	if dsn == "" {
		t.Fatal("dbtest: admin DSN not initialized. Call dbtest.RunMainWithPostgres(m) in TestMain.")
	}

	adminConn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("dbtest: open admin: %v", err)
	}
	defer func() { _ = adminConn.Close() }()

	dbName := fmt.Sprintf("t_%d_%d", time.Now().UnixNano(), dbCounter.Add(1))
	if _, err := adminConn.ExecContext(t.Context(), fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)); err != nil {
		t.Fatalf("dbtest: create database: %v", err)
	}
	t.Cleanup(func() {
		closeConn, err := sql.Open("pgx", dsn)
		if err != nil {
			t.Logf("dbtest: reopen admin to drop: %v", err)
			return
		}
		defer func() { _ = closeConn.Close() }()
		_, _ = closeConn.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, dbName))
	})

	testDSN := withDatabaseName(dsn, dbName)
	db, err := sql.Open("pgx", testDSN)
	if err != nil {
		t.Fatalf("dbtest: open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	if err := infradb.RunGoose(ctx, db, "up"); err != nil {
		t.Fatalf("dbtest: goose up: %v", err)
	}
	return db
}

func setup(ctx context.Context) error {
	mu.Lock()
	defer mu.Unlock()

	if dsn := os.Getenv(envTestDBDSN); dsn != "" {
		adminDSN = dsn
		return nil
	}

	pgC, err := tcpostgres.Run(ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase("postgres"),
		tcpostgres.WithUsername("appuser"),
		tcpostgres.WithPassword("apppass"),
		testcontainers.WithWaitStrategyAndDeadline(60*time.Second),
	)
	if err != nil {
		return fmt.Errorf("dbtest: start postgres container: %w", err)
	}
	container = pgC
	dsn, err := pgC.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("dbtest: get DSN: %w", err)
	}
	adminDSN = dsn
	return nil
}

func teardown() {
	mu.Lock()
	defer mu.Unlock()
	if container == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = container.Terminate(ctx)
}

// withDatabaseName は postgres URL の DB 名部分を差し替えます。
// 入力例: postgres://user:pass@host:5432/postgres?sslmode=disable
// 出力例: postgres://user:pass@host:5432/<dbName>?sslmode=disable
func withDatabaseName(dsn, dbName string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		// URL 形式でない場合は key=value 形式とみなし、dbname=... を差し替えるか追記する。
		return replaceOrAppendKeyValue(dsn, "dbname", dbName)
	}
	u.Path = "/" + dbName
	return u.String()
}

func replaceOrAppendKeyValue(dsn, key, value string) string {
	parts := strings.Fields(dsn)
	replaced := false
	for i, p := range parts {
		if strings.HasPrefix(p, key+"=") {
			parts[i] = fmt.Sprintf("%s=%s", key, value)
			replaced = true
		}
	}
	if !replaced {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(parts, " ")
}
