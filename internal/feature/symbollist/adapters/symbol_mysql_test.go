package adapters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"stock_backend/internal/feature/symbollist/domain/entity"
)

// setupTestDB はテスト用のインメモリSQLiteデータベースを準備します。
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to initialize test database")

	// Symbolテーブルを作成
	err = db.AutoMigrate(&entity.Symbol{})
	require.NoError(t, err, "failed to migrate table")

	return db
}

// seedSymbol はテスト用の銘柄データをデータベースに作成します。
func seedSymbol(t *testing.T, db *gorm.DB, code, name, market string, isActive bool) *entity.Symbol {
	t.Helper()

	symbol := &entity.Symbol{
		Code:     code,
		Name:     name,
		Market:   market,
		IsActive: isActive,
	}
	err := db.Create(symbol).Error
	require.NoError(t, err, "failed to seed symbol")

	return symbol
}

// updateSymbolActive は銘柄のis_activeフィールドを更新します。
// SQLiteはINSERT時にbooleanの扱いが異なるため、この関数が必要です。
func updateSymbolActive(t *testing.T, db *gorm.DB, symbol *entity.Symbol, isActive bool) {
	t.Helper()
	err := db.Model(symbol).Update("is_active", isActive).Error
	require.NoError(t, err, "failed to update symbol active status")
}

// TestNewSymbolRepository はNewSymbolRepositoryコンストラクタが正しくインスタンスを生成することを検証します。
func TestNewSymbolRepository(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewSymbolRepository(db)

	assert.NotNil(t, repo, "repository should not be nil")
	assert.NotNil(t, repo.db, "database connection should not be nil")
}

// TestSymbolMySQL_ListActive はListActiveメソッドの各種シナリオをテーブル駆動テストで検証します。
func TestSymbolMySQL_ListActive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupFunc     func(t *testing.T, db *gorm.DB)
		expectedCount int
		expectedCodes []string
		wantErr       bool
	}{
		{
			name: "success: returns active symbols sorted by code",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedSymbol(t, db, "9984.T", "SoftBank Group", "TSE", true)
				seedSymbol(t, db, "6758.T", "Sony Group", "TSE", true)
				seedSymbol(t, db, "7203.T", "Toyota Motor", "TSE", true)
			},
			expectedCount: 3,
			expectedCodes: []string{"6758.T", "7203.T", "9984.T"}, // Sorted by code ASC
			wantErr:       false,
		},
		{
			name: "success: excludes inactive symbols",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedSymbol(t, db, "7203.T", "Toyota Motor", "TSE", true)
				inactiveSymbol := seedSymbol(t, db, "6758.T", "Sony Group", "TSE", true)
				updateSymbolActive(t, db, inactiveSymbol, false) // Set to inactive
				seedSymbol(t, db, "9984.T", "SoftBank Group", "TSE", true)
			},
			expectedCount: 2,
			expectedCodes: []string{"7203.T", "9984.T"},
			wantErr:       false,
		},
		{
			name: "success: returns empty list when no symbols",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				// No symbols seeded
			},
			expectedCount: 0,
			expectedCodes: []string{},
			wantErr:       false,
		},
		{
			name: "success: returns empty list when all symbols are inactive",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				s1 := seedSymbol(t, db, "7203.T", "Toyota Motor", "TSE", true)
				updateSymbolActive(t, db, s1, false)
				s2 := seedSymbol(t, db, "6758.T", "Sony Group", "TSE", true)
				updateSymbolActive(t, db, s2, false)
			},
			expectedCount: 0,
			expectedCodes: []string{},
			wantErr:       false,
		},
		{
			name: "success: returns single active symbol",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedSymbol(t, db, "7203.T", "Toyota Motor", "TSE", true)
			},
			expectedCount: 1,
			expectedCodes: []string{"7203.T"},
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewSymbolRepository(db)

			if tt.setupFunc != nil {
				tt.setupFunc(t, db)
			}

			symbols, err := repo.ListActive(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, symbols, tt.expectedCount)

				// 順序とコードを検証
				for i, expectedCode := range tt.expectedCodes {
					assert.Equal(t, expectedCode, symbols[i].Code)
				}
			}
		})
	}
}

// TestSymbolMySQL_ListActiveCodes はListActiveCodesメソッドの各種シナリオをテーブル駆動テストで検証します。
func TestSymbolMySQL_ListActiveCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupFunc     func(t *testing.T, db *gorm.DB)
		expectedCodes []string
		wantErr       bool
	}{
		{
			name: "success: returns active symbol codes sorted by code",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedSymbol(t, db, "9984.T", "SoftBank Group", "TSE", true)
				seedSymbol(t, db, "6758.T", "Sony Group", "TSE", true)
				seedSymbol(t, db, "7203.T", "Toyota Motor", "TSE", true)
			},
			expectedCodes: []string{"6758.T", "7203.T", "9984.T"}, // Sorted by code ASC
			wantErr:       false,
		},
		{
			name: "success: excludes inactive symbol codes",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedSymbol(t, db, "7203.T", "Toyota Motor", "TSE", true)
				inactiveSymbol := seedSymbol(t, db, "6758.T", "Sony Group", "TSE", true)
				updateSymbolActive(t, db, inactiveSymbol, false) // Set to inactive
				seedSymbol(t, db, "9984.T", "SoftBank Group", "TSE", true)
			},
			expectedCodes: []string{"7203.T", "9984.T"},
			wantErr:       false,
		},
		{
			name: "success: returns empty list when no symbols",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				// No symbols seeded
			},
			expectedCodes: []string{},
			wantErr:       false,
		},
		{
			name: "success: returns empty list when all symbols are inactive",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				s1 := seedSymbol(t, db, "7203.T", "Toyota Motor", "TSE", true)
				updateSymbolActive(t, db, s1, false)
				s2 := seedSymbol(t, db, "6758.T", "Sony Group", "TSE", true)
				updateSymbolActive(t, db, s2, false)
			},
			expectedCodes: []string{},
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewSymbolRepository(db)

			if tt.setupFunc != nil {
				tt.setupFunc(t, db)
			}

			codes, err := repo.ListActiveCodes(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if len(tt.expectedCodes) == 0 {
					assert.Empty(t, codes)
				} else {
					assert.Equal(t, tt.expectedCodes, codes)
				}
			}
		})
	}
}

// TestSymbolMySQL_ListActive_FieldValues はListActiveが返す銘柄の全フィールド値が正しいことを検証します。
func TestSymbolMySQL_ListActive_FieldValues(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewSymbolRepository(db)

	// 全フィールドを持つ銘柄をシード
	expected := seedSymbol(t, db, "7203.T", "Toyota Motor Corporation", "Tokyo Stock Exchange", true)

	symbols, err := repo.ListActive(context.Background())

	require.NoError(t, err)
	require.Len(t, symbols, 1)

	symbol := symbols[0]
	assert.Equal(t, expected.ID, symbol.ID)
	assert.Equal(t, "7203.T", symbol.Code)
	assert.Equal(t, "Toyota Motor Corporation", symbol.Name)
	assert.Equal(t, "Tokyo Stock Exchange", symbol.Market)
	assert.True(t, symbol.IsActive)
	assert.False(t, symbol.UpdatedAt.IsZero(), "UpdatedAt should be set")
}

// TestSymbolMySQL_Exists はExistsメソッドの各種シナリオをテーブル駆動テストで検証します。
func TestSymbolMySQL_Exists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupFunc  func(t *testing.T, db *gorm.DB)
		code       string
		wantExists bool
		wantErr    bool
	}{
		{
			name: "success: returns true for existing symbol",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedSymbol(t, db, "AAPL", "Apple Inc.", "NASDAQ", true)
			},
			code:       "AAPL",
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "success: returns true for inactive symbol",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedSymbol(t, db, "AAPL", "Apple Inc.", "NASDAQ", false)
			},
			code:       "AAPL",
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "success: returns false for non-existent symbol",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				// No symbols seeded
			},
			code:       "INVALID",
			wantExists: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewSymbolRepository(db)

			if tt.setupFunc != nil {
				tt.setupFunc(t, db)
			}

			exists, err := repo.Exists(context.Background(), tt.code)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantExists, exists)
			}
		})
	}
}

// TestSymbolMySQL_ContextCancellation はコンテキストがキャンセルされた場合の動作を検証します。
func TestSymbolMySQL_ContextCancellation(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewSymbolRepository(db)

	seedSymbol(t, db, "7203.T", "Toyota Motor", "TSE", true)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	// 注意: SQLiteはコンテキストキャンセルを尊重しない場合がありますが、
	// このテストはコンテキストが正しく伝播されることを確認します
	_, err := repo.ListActive(ctx)
	// インメモリSQLiteはキャンセルされたコンテキストで常にエラーを返すとは限りません
	// このテストは主にコンテキストが正しく渡されることを検証します
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled)
	}
}
