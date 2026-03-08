package adapters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"stock_backend/internal/feature/watchlist/domain/entity"
	"stock_backend/internal/feature/watchlist/usecase"
)

// setupTestDB はテスト用のインメモリSQLiteデータベースを準備します。
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to initialize test database")

	err = db.AutoMigrate(&entity.UserSymbol{})
	require.NoError(t, err, "failed to migrate table")

	return db
}

// seedUserSymbol はテスト用のウォッチリスト銘柄をデータベースに作成します。
func seedUserSymbol(t *testing.T, db *gorm.DB, userID uint, symbolCode string, sortKey int) *entity.UserSymbol {
	t.Helper()

	us := &entity.UserSymbol{
		UserID:     userID,
		SymbolCode: symbolCode,
		SortKey:    sortKey,
	}
	err := db.Create(us).Error
	require.NoError(t, err, "failed to seed user symbol")

	return us
}

func TestNewUserSymbolRepository(t *testing.T) {
	db := setupTestDB(t)

	repo := NewUserSymbolRepository(db)

	assert.NotNil(t, repo, "repository is nil")
	assert.NotNil(t, repo.db, "database connection is nil")
}

func TestUserSymbolMySQL_ListByUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		userID       uint
		setupFunc    func(t *testing.T, db *gorm.DB)
		validateFunc func(t *testing.T, symbols []entity.UserSymbol)
	}{
		{
			name:   "success: returns symbols ordered by sort_key",
			userID: 1,
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedUserSymbol(t, db, 1, "GOOGL", 30)
				seedUserSymbol(t, db, 1, "AAPL", 10)
				seedUserSymbol(t, db, 1, "MSFT", 20)
			},
			validateFunc: func(t *testing.T, symbols []entity.UserSymbol) {
				require.Len(t, symbols, 3)
				assert.Equal(t, "AAPL", symbols[0].SymbolCode)
				assert.Equal(t, "MSFT", symbols[1].SymbolCode)
				assert.Equal(t, "GOOGL", symbols[2].SymbolCode)
			},
		},
		{
			name:   "success: returns empty slice when no symbols",
			userID: 1,
			validateFunc: func(t *testing.T, symbols []entity.UserSymbol) {
				assert.Empty(t, symbols)
			},
		},
		{
			name:   "success: returns only symbols for specified user",
			userID: 1,
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedUserSymbol(t, db, 1, "AAPL", 10)
				seedUserSymbol(t, db, 2, "MSFT", 10)
				seedUserSymbol(t, db, 1, "GOOGL", 20)
			},
			validateFunc: func(t *testing.T, symbols []entity.UserSymbol) {
				require.Len(t, symbols, 2)
				assert.Equal(t, "AAPL", symbols[0].SymbolCode)
				assert.Equal(t, "GOOGL", symbols[1].SymbolCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewUserSymbolRepository(db)

			if tt.setupFunc != nil {
				tt.setupFunc(t, db)
			}

			symbols, err := repo.ListByUser(context.Background(), tt.userID)

			assert.NoError(t, err)
			if tt.validateFunc != nil {
				tt.validateFunc(t, symbols)
			}
		})
	}
}

func TestUserSymbolMySQL_AddWithAtomicSortKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		userID      uint
		symbolCode  string
		setupFunc   func(t *testing.T, db *gorm.DB)
		wantErr     bool
		wantSortKey int
	}{
		{
			name:        "success: first symbol gets sort_key 10",
			userID:      1,
			symbolCode:  "AAPL",
			wantSortKey: 10,
		},
		{
			name:       "success: second symbol gets max+10",
			userID:     1,
			symbolCode: "MSFT",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedUserSymbol(t, db, 1, "AAPL", 10)
			},
			wantSortKey: 20,
		},
		{
			name:       "error: duplicate user-symbol pair",
			userID:     1,
			symbolCode: "AAPL",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedUserSymbol(t, db, 1, "AAPL", 10)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewUserSymbolRepository(db)

			if tt.setupFunc != nil {
				tt.setupFunc(t, db)
			}

			err := repo.AddWithAtomicSortKey(context.Background(), tt.userID, tt.symbolCode)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				var inserted entity.UserSymbol
				require.NoError(t, db.Where("user_id = ? AND symbol_code = ?", tt.userID, tt.symbolCode).First(&inserted).Error)
				assert.Equal(t, tt.wantSortKey, inserted.SortKey)
			}
		})
	}
}

func TestUserSymbolMySQL_Remove(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		userID      uint
		symbolCode  string
		wantErr     bool
		expectedErr error
		setupFunc   func(t *testing.T, db *gorm.DB)
	}{
		{
			name:       "success: remove existing symbol",
			userID:     1,
			symbolCode: "AAPL",
			wantErr:    false,
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedUserSymbol(t, db, 1, "AAPL", 10)
			},
		},
		{
			name:        "error: symbol not found",
			userID:      1,
			symbolCode:  "AAPL",
			wantErr:     true,
			expectedErr: usecase.ErrSymbolNotFound,
		},
		{
			name:        "error: wrong user ID",
			userID:      2,
			symbolCode:  "AAPL",
			wantErr:     true,
			expectedErr: usecase.ErrSymbolNotFound,
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedUserSymbol(t, db, 1, "AAPL", 10)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewUserSymbolRepository(db)

			if tt.setupFunc != nil {
				tt.setupFunc(t, db)
			}

			err := repo.Remove(context.Background(), tt.userID, tt.symbolCode)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				assert.NoError(t, err)

				// verify symbol is removed
				var count int64
				db.Model(&entity.UserSymbol{}).
					Where("user_id = ? AND symbol_code = ?", tt.userID, tt.symbolCode).
					Count(&count)
				assert.Equal(t, int64(0), count, "symbol should be removed")
			}
		})
	}
}

func TestUserSymbolMySQL_UpdateSortKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		userID       uint
		codeOrder    []string
		setupFunc    func(t *testing.T, db *gorm.DB)
		validateFunc func(t *testing.T, db *gorm.DB)
	}{
		{
			name:      "success: reorder symbols",
			userID:    1,
			codeOrder: []string{"GOOGL", "AAPL", "MSFT"},
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedUserSymbol(t, db, 1, "AAPL", 10)
				seedUserSymbol(t, db, 1, "MSFT", 20)
				seedUserSymbol(t, db, 1, "GOOGL", 30)
			},
			validateFunc: func(t *testing.T, db *gorm.DB) {
				var symbols []entity.UserSymbol
				err := db.Where("user_id = ?", 1).Order("sort_key ASC").Find(&symbols).Error
				require.NoError(t, err)
				require.Len(t, symbols, 3)
				assert.Equal(t, "GOOGL", symbols[0].SymbolCode)
				assert.Equal(t, 0, symbols[0].SortKey)
				assert.Equal(t, "AAPL", symbols[1].SymbolCode)
				assert.Equal(t, 10, symbols[1].SortKey)
				assert.Equal(t, "MSFT", symbols[2].SymbolCode)
				assert.Equal(t, 20, symbols[2].SortKey)
			},
		},
		{
			name:      "success: empty code order",
			userID:    1,
			codeOrder: []string{},
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedUserSymbol(t, db, 1, "AAPL", 10)
			},
			validateFunc: func(t *testing.T, db *gorm.DB) {
				var us entity.UserSymbol
				err := db.Where("user_id = ? AND symbol_code = ?", 1, "AAPL").First(&us).Error
				require.NoError(t, err)
				assert.Equal(t, 10, us.SortKey, "sort_key should remain unchanged")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewUserSymbolRepository(db)

			if tt.setupFunc != nil {
				tt.setupFunc(t, db)
			}

			err := repo.UpdateSortKeys(context.Background(), tt.userID, tt.codeOrder)

			assert.NoError(t, err)
			if tt.validateFunc != nil {
				tt.validateFunc(t, db)
			}
		})
	}
}

func TestUserSymbolMySQL_AddBatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		symbols      []entity.UserSymbol
		wantErr      bool
		setupFunc    func(t *testing.T, db *gorm.DB)
		validateFunc func(t *testing.T, db *gorm.DB)
	}{
		{
			name: "success: add multiple symbols",
			symbols: []entity.UserSymbol{
				{UserID: 1, SymbolCode: "AAPL", SortKey: 10},
				{UserID: 1, SymbolCode: "MSFT", SortKey: 20},
				{UserID: 1, SymbolCode: "GOOGL", SortKey: 30},
			},
			validateFunc: func(t *testing.T, db *gorm.DB) {
				var count int64
				db.Model(&entity.UserSymbol{}).Where("user_id = ?", 1).Count(&count)
				assert.Equal(t, int64(3), count)
			},
		},
		{
			name:    "success: empty slice does nothing",
			symbols: []entity.UserSymbol{},
			validateFunc: func(t *testing.T, db *gorm.DB) {
				var count int64
				db.Model(&entity.UserSymbol{}).Count(&count)
				assert.Equal(t, int64(0), count)
			},
		},
		{
			// NOTE: AddBatch's duplicate-ignore logic relies on MySQL error code 1062.
			// SQLite returns a different error type, so duplicates cause an error here.
			name: "error: duplicate symbols return error with non-MySQL driver",
			symbols: []entity.UserSymbol{
				{UserID: 1, SymbolCode: "AAPL", SortKey: 20},
				{UserID: 1, SymbolCode: "MSFT", SortKey: 30},
			},
			wantErr: true,
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedUserSymbol(t, db, 1, "AAPL", 10)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewUserSymbolRepository(db)

			if tt.setupFunc != nil {
				tt.setupFunc(t, db)
			}

			err := repo.AddBatch(context.Background(), tt.symbols)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.validateFunc != nil {
				tt.validateFunc(t, db)
			}
		})
	}
}

func TestUserSymbolMySQL_MaxSortKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		userID    uint
		want      int
		setupFunc func(t *testing.T, db *gorm.DB)
	}{
		{
			name:   "success: returns max sort_key",
			userID: 1,
			want:   30,
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedUserSymbol(t, db, 1, "AAPL", 10)
				seedUserSymbol(t, db, 1, "MSFT", 30)
				seedUserSymbol(t, db, 1, "GOOGL", 20)
			},
		},
		{
			name:   "success: returns 0 when no symbols",
			userID: 1,
			want:   0,
		},
		{
			name:   "success: returns max for specific user only",
			userID: 1,
			want:   10,
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedUserSymbol(t, db, 1, "AAPL", 10)
				seedUserSymbol(t, db, 2, "MSFT", 50)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewUserSymbolRepository(db)

			if tt.setupFunc != nil {
				tt.setupFunc(t, db)
			}

			got, err := repo.MaxSortKey(context.Background(), tt.userID)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
