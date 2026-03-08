package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"stock_backend/internal/feature/watchlist/domain/entity"
	"stock_backend/internal/feature/watchlist/usecase"
)

// mockUserSymbolRepository はUserSymbolRepositoryインターフェースのモック実装です。
type mockUserSymbolRepository struct {
	ListByUserFunc     func(ctx context.Context, userID uint) ([]entity.UserSymbol, error)
	AddFunc            func(ctx context.Context, us *entity.UserSymbol) error
	RemoveFunc         func(ctx context.Context, userID uint, symbolCode string) error
	UpdateSortKeysFunc func(ctx context.Context, userID uint, codeOrder []string) error
	AddBatchFunc       func(ctx context.Context, userSymbols []entity.UserSymbol) error
	MaxSortKeyFunc     func(ctx context.Context, userID uint) (int, error)
}

func (m *mockUserSymbolRepository) ListByUser(ctx context.Context, userID uint) ([]entity.UserSymbol, error) {
	if m.ListByUserFunc != nil {
		return m.ListByUserFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockUserSymbolRepository) Add(ctx context.Context, us *entity.UserSymbol) error {
	if m.AddFunc != nil {
		return m.AddFunc(ctx, us)
	}
	return nil
}

func (m *mockUserSymbolRepository) Remove(ctx context.Context, userID uint, symbolCode string) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(ctx, userID, symbolCode)
	}
	return nil
}

func (m *mockUserSymbolRepository) UpdateSortKeys(ctx context.Context, userID uint, codeOrder []string) error {
	if m.UpdateSortKeysFunc != nil {
		return m.UpdateSortKeysFunc(ctx, userID, codeOrder)
	}
	return nil
}

func (m *mockUserSymbolRepository) AddBatch(ctx context.Context, userSymbols []entity.UserSymbol) error {
	if m.AddBatchFunc != nil {
		return m.AddBatchFunc(ctx, userSymbols)
	}
	return nil
}

func (m *mockUserSymbolRepository) MaxSortKey(ctx context.Context, userID uint) (int, error) {
	if m.MaxSortKeyFunc != nil {
		return m.MaxSortKeyFunc(ctx, userID)
	}
	return 0, nil
}

func TestWatchlistUsecase_ListUserSymbols(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		userID   uint
		mockList func(ctx context.Context, userID uint) ([]entity.UserSymbol, error)
		want     []entity.UserSymbol
		wantErr  bool
	}{
		{
			name:   "success: returns user symbols",
			userID: 1,
			mockList: func(ctx context.Context, userID uint) ([]entity.UserSymbol, error) {
				return []entity.UserSymbol{
					{ID: 1, UserID: 1, SymbolCode: "AAPL", SortKey: 10},
					{ID: 2, UserID: 1, SymbolCode: "MSFT", SortKey: 20},
				}, nil
			},
			want: []entity.UserSymbol{
				{ID: 1, UserID: 1, SymbolCode: "AAPL", SortKey: 10},
				{ID: 2, UserID: 1, SymbolCode: "MSFT", SortKey: 20},
			},
		},
		{
			name:   "success: empty watchlist",
			userID: 1,
			mockList: func(ctx context.Context, userID uint) ([]entity.UserSymbol, error) {
				return []entity.UserSymbol{}, nil
			},
			want: []entity.UserSymbol{},
		},
		{
			name:   "failure: repository error",
			userID: 1,
			mockList: func(ctx context.Context, userID uint) ([]entity.UserSymbol, error) {
				return nil, errors.New("database error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &mockUserSymbolRepository{ListByUserFunc: tt.mockList}
			uc := usecase.NewWatchlistUsecase(repo)

			got, err := uc.ListUserSymbols(context.Background(), tt.userID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestWatchlistUsecase_AddSymbol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     uint
		symbolCode string
		maxSortKey int
		addErr     error
		wantErr    bool
		errIs      error
	}{
		{
			name:       "success: add symbol",
			userID:     1,
			symbolCode: "TSLA",
			maxSortKey: 20,
		},
		{
			name:       "failure: empty symbol code",
			userID:     1,
			symbolCode: "",
			wantErr:    true,
		},
		{
			name:       "failure: duplicate symbol",
			userID:     1,
			symbolCode: "AAPL",
			addErr:     usecase.ErrSymbolAlreadyExists,
			wantErr:    true,
			errIs:      usecase.ErrSymbolAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var addedSymbol *entity.UserSymbol
			repo := &mockUserSymbolRepository{
				MaxSortKeyFunc: func(ctx context.Context, userID uint) (int, error) {
					return tt.maxSortKey, nil
				},
				AddFunc: func(ctx context.Context, us *entity.UserSymbol) error {
					addedSymbol = us
					return tt.addErr
				},
			}
			uc := usecase.NewWatchlistUsecase(repo)

			err := uc.AddSymbol(context.Background(), tt.userID, tt.symbolCode)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errIs != nil {
					assert.ErrorIs(t, err, tt.errIs)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.userID, addedSymbol.UserID)
				assert.Equal(t, tt.symbolCode, addedSymbol.SymbolCode)
				assert.Equal(t, tt.maxSortKey+10, addedSymbol.SortKey)
			}
		})
	}
}

func TestWatchlistUsecase_RemoveSymbol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		removeErr error
		wantErr   bool
		errIs     error
	}{
		{
			name: "success: remove symbol",
		},
		{
			name:      "failure: symbol not found",
			removeErr: usecase.ErrSymbolNotFound,
			wantErr:   true,
			errIs:     usecase.ErrSymbolNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &mockUserSymbolRepository{
				RemoveFunc: func(ctx context.Context, userID uint, symbolCode string) error {
					return tt.removeErr
				},
			}
			uc := usecase.NewWatchlistUsecase(repo)

			err := uc.RemoveSymbol(context.Background(), 1, "AAPL")
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errIs != nil {
					assert.ErrorIs(t, err, tt.errIs)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWatchlistUsecase_ReorderSymbols(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		codeOrder []string
		repoErr   error
		wantErr   bool
	}{
		{
			name:      "success: reorder symbols",
			codeOrder: []string{"MSFT", "AAPL", "GOOGL"},
		},
		{
			name:      "failure: empty code order",
			codeOrder: []string{},
			wantErr:   true,
		},
		{
			name:      "failure: repository error",
			codeOrder: []string{"AAPL"},
			repoErr:   errors.New("database error"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &mockUserSymbolRepository{
				UpdateSortKeysFunc: func(ctx context.Context, userID uint, codeOrder []string) error {
					return tt.repoErr
				},
			}
			uc := usecase.NewWatchlistUsecase(repo)

			err := uc.ReorderSymbols(context.Background(), 1, tt.codeOrder)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWatchlistUsecase_InitializeDefaults(t *testing.T) {
	t.Parallel()

	t.Run("success: creates default symbols", func(t *testing.T) {
		t.Parallel()

		var addedSymbols []entity.UserSymbol
		repo := &mockUserSymbolRepository{
			AddBatchFunc: func(ctx context.Context, userSymbols []entity.UserSymbol) error {
				addedSymbols = userSymbols
				return nil
			},
		}
		uc := usecase.NewWatchlistUsecase(repo)

		err := uc.InitializeDefaults(context.Background(), 42)
		require.NoError(t, err)

		assert.Len(t, addedSymbols, len(usecase.DefaultSymbolCodes))
		for i, code := range usecase.DefaultSymbolCodes {
			assert.Equal(t, uint(42), addedSymbols[i].UserID)
			assert.Equal(t, code, addedSymbols[i].SymbolCode)
			assert.Equal(t, (i+1)*10, addedSymbols[i].SortKey)
		}
	})

	t.Run("failure: repository error", func(t *testing.T) {
		t.Parallel()

		repo := &mockUserSymbolRepository{
			AddBatchFunc: func(ctx context.Context, userSymbols []entity.UserSymbol) error {
				return errors.New("database error")
			},
		}
		uc := usecase.NewWatchlistUsecase(repo)

		err := uc.InitializeDefaults(context.Background(), 1)
		assert.Error(t, err)
	})
}
