package cache

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"

	"stock_backend/internal/feature/candles/domain/entity"
)

// mockCandleRepository はテスト用のCandleRepositoryモック実装です。
type mockCandleRepository struct {
	findFn        func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
	upsertBatchFn func(ctx context.Context, candles []entity.Candle) error
}

// Find はモックのFind関数を呼び出します。
func (m *mockCandleRepository) Find(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	if m.findFn != nil {
		return m.findFn(ctx, symbol, interval, outputsize)
	}
	return nil, nil
}

// UpsertBatch はモックのUpsertBatch関数を呼び出します。
func (m *mockCandleRepository) UpsertBatch(ctx context.Context, candles []entity.Candle) error {
	if m.upsertBatchFn != nil {
		return m.upsertBatchFn(ctx, candles)
	}
	return nil
}

// TestNewCachingCandleRepository_Defaults はデフォルト値（TTLとnamespace）が正しく設定されることを検証します。
func TestNewCachingCandleRepository_Defaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		ttl               time.Duration
		namespace         string
		expectedTTL       time.Duration
		expectedNamespace string
	}{
		{
			name:              "default values when zero/empty",
			ttl:               0,
			namespace:         "",
			expectedTTL:       5 * time.Minute,
			expectedNamespace: "candles",
		},
		{
			name:              "negative ttl uses default",
			ttl:               -1 * time.Minute,
			namespace:         "",
			expectedTTL:       5 * time.Minute,
			expectedNamespace: "candles",
		},
		{
			name:              "custom values preserved",
			ttl:               10 * time.Minute,
			namespace:         "custom",
			expectedTTL:       10 * time.Minute,
			expectedNamespace: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := NewCachingCandleRepository(nil, tt.ttl, &mockCandleRepository{}, tt.namespace)

			if repo.ttl != tt.expectedTTL {
				t.Errorf("expected TTL %v, got %v", tt.expectedTTL, repo.ttl)
			}
			if repo.namespace != tt.expectedNamespace {
				t.Errorf("expected namespace %q, got %q", tt.expectedNamespace, repo.namespace)
			}
		})
	}
}

// TestCachingCandleRepository_Find_NilRedis はRedisがnilの場合にキャッシュをバイパスして内部リポジトリを直接呼び出すことを検証します。
func TestCachingCandleRepository_Find_NilRedis(t *testing.T) {
	t.Parallel()

	expectedCandles := []entity.Candle{
		{Symbol: "AAPL", Interval: "1day", Open: 150.0, Close: 155.0},
	}

	inner := &mockCandleRepository{
		findFn: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
			return expectedCandles, nil
		},
	}

	// Redis is nil - should bypass cache and call inner directly
	repo := NewCachingCandleRepository(nil, 5*time.Minute, inner, "candles")

	candles, err := repo.Find(context.Background(), "AAPL", "1day", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candles) != len(expectedCandles) {
		t.Errorf("expected %d candles, got %d", len(expectedCandles), len(candles))
	}
}

// TestCachingCandleRepository_Find_CacheHit はキャッシュヒット時にRedisからデータを返し、内部リポジトリを呼ばないことを検証します。
func TestCachingCandleRepository_Find_CacheHit(t *testing.T) {
	t.Parallel()

	rdb, mock := redismock.NewClientMock()
	defer func() { _ = rdb.Close() }()

	cachedCandles := []entity.Candle{
		{Symbol: "AAPL", Interval: "1day", Open: 150.0, Close: 155.0},
	}
	cachedJSON, _ := json.Marshal(cachedCandles)

	mock.ExpectGet("candles:AAPL:1day:100").SetVal(string(cachedJSON))

	innerCalled := false
	inner := &mockCandleRepository{
		findFn: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
			innerCalled = true
			return nil, nil
		},
	}

	repo := NewCachingCandleRepository(rdb, 5*time.Minute, inner, "candles")
	candles, err := repo.Find(context.Background(), "AAPL", "1day", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if innerCalled {
		t.Error("inner repository should not be called on cache hit")
	}
	if len(candles) != 1 {
		t.Errorf("expected 1 candle, got %d", len(candles))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled mock expectations: %v", err)
	}
}

// TestCachingCandleRepository_Find_CacheMiss はキャッシュミス時にDBからデータを取得し、キャッシュに保存することを検証します。
func TestCachingCandleRepository_Find_CacheMiss(t *testing.T) {
	t.Parallel()

	rdb, mock := redismock.NewClientMock()
	defer func() { _ = rdb.Close() }()

	expectedCandles := []entity.Candle{
		{Symbol: "AAPL", Interval: "1day", Open: 150.0, Close: 155.0},
	}
	expectedJSON, _ := json.Marshal(expectedCandles)

	// Cache miss
	mock.ExpectGet("candles:AAPL:1day:100").RedisNil()
	// Set cache after fetching from inner
	mock.ExpectSet("candles:AAPL:1day:100", expectedJSON, 5*time.Minute).SetVal("OK")

	inner := &mockCandleRepository{
		findFn: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
			return expectedCandles, nil
		},
	}

	repo := NewCachingCandleRepository(rdb, 5*time.Minute, inner, "candles")
	candles, err := repo.Find(context.Background(), "AAPL", "1day", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candles) != 1 {
		t.Errorf("expected 1 candle, got %d", len(candles))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled mock expectations: %v", err)
	}
}

// TestCachingCandleRepository_Find_InnerError は内部リポジトリがエラーを返した場合にそのエラーが伝播されることを検証します。
func TestCachingCandleRepository_Find_InnerError(t *testing.T) {
	t.Parallel()

	rdb, mock := redismock.NewClientMock()
	defer func() { _ = rdb.Close() }()

	expectedErr := errors.New("database error")

	mock.ExpectGet("candles:AAPL:1day:100").RedisNil()

	inner := &mockCandleRepository{
		findFn: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
			return nil, expectedErr
		},
	}

	repo := NewCachingCandleRepository(rdb, 5*time.Minute, inner, "candles")
	_, err := repo.Find(context.Background(), "AAPL", "1day", 100)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

// TestCachingCandleRepository_Find_CorruptedCache は破損したキャッシュを検出・削除し、DBにフォールバックすることを検証します。
func TestCachingCandleRepository_Find_CorruptedCache(t *testing.T) {
	t.Parallel()

	rdb, mock := redismock.NewClientMock()
	defer func() { _ = rdb.Close() }()

	expectedCandles := []entity.Candle{
		{Symbol: "AAPL", Interval: "1day", Open: 150.0, Close: 155.0},
	}
	expectedJSON, _ := json.Marshal(expectedCandles)

	// Return invalid JSON from cache
	mock.ExpectGet("candles:AAPL:1day:100").SetVal("invalid json")
	// Delete corrupted cache
	mock.ExpectDel("candles:AAPL:1day:100").SetVal(1)
	// Set new cache after fetching from inner
	mock.ExpectSet("candles:AAPL:1day:100", expectedJSON, 5*time.Minute).SetVal("OK")

	inner := &mockCandleRepository{
		findFn: func(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
			return expectedCandles, nil
		},
	}

	repo := NewCachingCandleRepository(rdb, 5*time.Minute, inner, "candles")
	candles, err := repo.Find(context.Background(), "AAPL", "1day", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candles) != 1 {
		t.Errorf("expected 1 candle, got %d", len(candles))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled mock expectations: %v", err)
	}
}

// TestCachingCandleRepository_UpsertBatch_NilRedis はRedisがnilの場合にUpsertBatchが内部リポジトリのみを呼び出すことを検証します。
func TestCachingCandleRepository_UpsertBatch_NilRedis(t *testing.T) {
	t.Parallel()

	innerCalled := false
	inner := &mockCandleRepository{
		upsertBatchFn: func(ctx context.Context, candles []entity.Candle) error {
			innerCalled = true
			return nil
		},
	}

	repo := NewCachingCandleRepository(nil, 5*time.Minute, inner, "candles")
	err := repo.UpsertBatch(context.Background(), []entity.Candle{
		{Symbol: "AAPL", Interval: "1day"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !innerCalled {
		t.Error("expected inner repository to be called")
	}
}

// TestCachingCandleRepository_UpsertBatch_InnerError は内部リポジトリのUpsertBatchエラーが伝播されることを検証します。
func TestCachingCandleRepository_UpsertBatch_InnerError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("upsert error")
	inner := &mockCandleRepository{
		upsertBatchFn: func(ctx context.Context, candles []entity.Candle) error {
			return expectedErr
		},
	}

	repo := NewCachingCandleRepository(nil, 5*time.Minute, inner, "candles")
	err := repo.UpsertBatch(context.Background(), []entity.Candle{
		{Symbol: "AAPL", Interval: "1day"},
	})

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

// TestCachingCandleRepository_UpsertBatch_EmptyCandles は空のローソク足データでUpsertBatchが正常に完了することを検証します。
func TestCachingCandleRepository_UpsertBatch_EmptyCandles(t *testing.T) {
	t.Parallel()

	rdb, _ := redismock.NewClientMock()
	defer func() { _ = rdb.Close() }()

	inner := &mockCandleRepository{
		upsertBatchFn: func(ctx context.Context, candles []entity.Candle) error {
			return nil
		},
	}

	repo := NewCachingCandleRepository(rdb, 5*time.Minute, inner, "candles")
	err := repo.UpsertBatch(context.Background(), []entity.Candle{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestCachingCandleRepository_UpsertBatch_CacheInvalidation はUpsertBatch後に関連するキャッシュが無効化されることを検証します。
func TestCachingCandleRepository_UpsertBatch_CacheInvalidation(t *testing.T) {
	t.Parallel()

	rdb, mock := redismock.NewClientMock()
	defer func() { _ = rdb.Close() }()

	inner := &mockCandleRepository{
		upsertBatchFn: func(ctx context.Context, candles []entity.Candle) error {
			return nil
		},
	}

	// Expect cache invalidation via SCAN and DEL
	mock.ExpectScan(0, "candles:AAPL:1day:*", 200).SetVal([]string{"candles:AAPL:1day:100", "candles:AAPL:1day:200"}, 0)
	mock.ExpectDel("candles:AAPL:1day:100", "candles:AAPL:1day:200").SetVal(2)

	repo := NewCachingCandleRepository(rdb, 5*time.Minute, inner, "candles")
	err := repo.UpsertBatch(context.Background(), []entity.Candle{
		{Symbol: "AAPL", Interval: "1day"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled mock expectations: %v", err)
	}
}

// TestCachingCandleRepository_UpsertBatch_DeduplicatesInvalidation は同一symbol+intervalのキャッシュ無効化が重複せず1回のみ実行されることを検証します。
func TestCachingCandleRepository_UpsertBatch_DeduplicatesInvalidation(t *testing.T) {
	t.Parallel()

	rdb, mock := redismock.NewClientMock()
	defer func() { _ = rdb.Close() }()

	inner := &mockCandleRepository{
		upsertBatchFn: func(ctx context.Context, candles []entity.Candle) error {
			return nil
		},
	}

	// Only expect one SCAN call for AAPL:1day despite multiple candles
	mock.ExpectScan(0, "candles:AAPL:1day:*", 200).SetVal([]string{}, 0)

	repo := NewCachingCandleRepository(rdb, 5*time.Minute, inner, "candles")
	err := repo.UpsertBatch(context.Background(), []entity.Candle{
		{Symbol: "AAPL", Interval: "1day", Time: time.Now()},
		{Symbol: "AAPL", Interval: "1day", Time: time.Now().Add(-24 * time.Hour)},
		{Symbol: "AAPL", Interval: "1day", Time: time.Now().Add(-48 * time.Hour)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled mock expectations: %v", err)
	}
}

// TestSafe はsafe関数がRedisキーで問題となる文字を正しくエスケープすることを検証します。
func TestSafe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"AAPL", "AAPL"},
		{"BRK A", "BRK_A"},
		{"key:value", "key_value"},
		{"a b:c", "a_b_c"},
		{"", ""},
		{"  ", "__"},
		{"::", "__"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			result := safe(tt.input)
			if result != tt.expected {
				t.Errorf("safe(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
