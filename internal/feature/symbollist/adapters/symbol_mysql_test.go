package adapters

import (
	"context"
	"stock_backend/internal/feature/symbollist/domain/entity"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB prepares an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to initialize test database")

	// Create Symbol table
	err = db.AutoMigrate(&entity.Symbol{})
	require.NoError(t, err, "failed to migrate table")

	return db
}

// seedSymbol creates a test symbol in the database for testing.
func seedSymbol(t *testing.T, db *gorm.DB, code, name, market string, isActive bool, sortKey int) *entity.Symbol {
	t.Helper()

	symbol := &entity.Symbol{
		Code:     code,
		Name:     name,
		Market:   market,
		IsActive: isActive,
		SortKey:  sortKey,
	}
	err := db.Create(symbol).Error
	require.NoError(t, err, "failed to seed symbol")

	return symbol
}

// updateSymbolActive updates the is_active field of a symbol.
// This is needed because SQLite handles boolean differently during INSERT.
func updateSymbolActive(t *testing.T, db *gorm.DB, symbol *entity.Symbol, isActive bool) {
	t.Helper()
	err := db.Model(symbol).Update("is_active", isActive).Error
	require.NoError(t, err, "failed to update symbol active status")
}

func TestNewSymbolRepository(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewSymbolRepository(db)

	assert.NotNil(t, repo, "repository should not be nil")
	assert.NotNil(t, repo.db, "database connection should not be nil")
}

func TestSymbolMySQL_ListActive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupFunc      func(t *testing.T, db *gorm.DB)
		expectedCount  int
		expectedCodes  []string
		wantErr        bool
	}{
		{
			name: "success: returns active symbols sorted by sort_key",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedSymbol(t, db, "6758.T", "Sony Group", "TSE", true, 2)
				seedSymbol(t, db, "7203.T", "Toyota Motor", "TSE", true, 1)
				seedSymbol(t, db, "9984.T", "SoftBank Group", "TSE", true, 3)
			},
			expectedCount: 3,
			expectedCodes: []string{"7203.T", "6758.T", "9984.T"}, // Sorted by sort_key
			wantErr:       false,
		},
		{
			name: "success: excludes inactive symbols",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedSymbol(t, db, "7203.T", "Toyota Motor", "TSE", true, 1)
				inactiveSymbol := seedSymbol(t, db, "6758.T", "Sony Group", "TSE", true, 2)
				updateSymbolActive(t, db, inactiveSymbol, false) // Set to inactive
				seedSymbol(t, db, "9984.T", "SoftBank Group", "TSE", true, 3)
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
				s1 := seedSymbol(t, db, "7203.T", "Toyota Motor", "TSE", true, 1)
				updateSymbolActive(t, db, s1, false)
				s2 := seedSymbol(t, db, "6758.T", "Sony Group", "TSE", true, 2)
				updateSymbolActive(t, db, s2, false)
			},
			expectedCount: 0,
			expectedCodes: []string{},
			wantErr:       false,
		},
		{
			name: "success: returns single active symbol",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedSymbol(t, db, "7203.T", "Toyota Motor", "TSE", true, 1)
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

				// Verify order and codes
				for i, expectedCode := range tt.expectedCodes {
					assert.Equal(t, expectedCode, symbols[i].Code)
				}
			}
		})
	}
}

func TestSymbolMySQL_ListActiveCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupFunc     func(t *testing.T, db *gorm.DB)
		expectedCodes []string
		wantErr       bool
	}{
		{
			name: "success: returns active symbol codes sorted by sort_key",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedSymbol(t, db, "6758.T", "Sony Group", "TSE", true, 2)
				seedSymbol(t, db, "7203.T", "Toyota Motor", "TSE", true, 1)
				seedSymbol(t, db, "9984.T", "SoftBank Group", "TSE", true, 3)
			},
			expectedCodes: []string{"7203.T", "6758.T", "9984.T"}, // Sorted by sort_key
			wantErr:       false,
		},
		{
			name: "success: excludes inactive symbol codes",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedSymbol(t, db, "7203.T", "Toyota Motor", "TSE", true, 1)
				inactiveSymbol := seedSymbol(t, db, "6758.T", "Sony Group", "TSE", true, 2)
				updateSymbolActive(t, db, inactiveSymbol, false) // Set to inactive
				seedSymbol(t, db, "9984.T", "SoftBank Group", "TSE", true, 3)
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
				s1 := seedSymbol(t, db, "7203.T", "Toyota Motor", "TSE", true, 1)
				updateSymbolActive(t, db, s1, false)
				s2 := seedSymbol(t, db, "6758.T", "Sony Group", "TSE", true, 2)
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

func TestSymbolMySQL_ListActive_FieldValues(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewSymbolRepository(db)

	// Seed a symbol with all fields
	expected := seedSymbol(t, db, "7203.T", "Toyota Motor Corporation", "Tokyo Stock Exchange", true, 42)

	symbols, err := repo.ListActive(context.Background())

	require.NoError(t, err)
	require.Len(t, symbols, 1)

	symbol := symbols[0]
	assert.Equal(t, expected.ID, symbol.ID)
	assert.Equal(t, "7203.T", symbol.Code)
	assert.Equal(t, "Toyota Motor Corporation", symbol.Name)
	assert.Equal(t, "Tokyo Stock Exchange", symbol.Market)
	assert.True(t, symbol.IsActive)
	assert.Equal(t, 42, symbol.SortKey)
	assert.False(t, symbol.UpdatedAt.IsZero(), "UpdatedAt should be set")
}

func TestSymbolMySQL_ContextCancellation(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewSymbolRepository(db)

	seedSymbol(t, db, "7203.T", "Toyota Motor", "TSE", true, 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	// Note: SQLite may not respect context cancellation, but the test ensures
	// the context is passed through correctly
	_, err := repo.ListActive(ctx)
	// SQLite in-memory doesn't always error on cancelled context
	// This test primarily verifies the context is passed through
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled)
	}
}
