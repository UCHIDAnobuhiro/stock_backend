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
	ListByUserFunc           func(ctx context.Context, userID uint) ([]entity.UserSymbol, error)
	AddWithAtomicSortKeyFunc func(ctx context.Context, userID uint, symbolCode string) error
	RemoveFunc               func(ctx context.Context, userID uint, symbolCode string) error
	UpdateSortKeysFunc       func(ctx context.Context, userID uint, codeOrder []string) error
	AddBatchFunc             func(ctx context.Context, userSymbols []entity.UserSymbol) error
}

func (m *mockUserSymbolRepository) ListByUser(ctx context.Context, userID uint) ([]entity.UserSymbol, error) {
	if m.ListByUserFunc != nil {
		return m.ListByUserFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockUserSymbolRepository) AddWithAtomicSortKey(ctx context.Context, userID uint, symbolCode string) error {
	if m.AddWithAtomicSortKeyFunc != nil {
		return m.AddWithAtomicSortKeyFunc(ctx, userID, symbolCode)
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

// mockSymbolChecker はSymbolCheckerインターフェースのモック実装です。
type mockSymbolChecker struct {
	ExistsCodeFunc func(ctx context.Context, code string) (bool, error)
}

func (m *mockSymbolChecker) ExistsCode(ctx context.Context, code string) (bool, error) {
	if m.ExistsCodeFunc != nil {
		return m.ExistsCodeFunc(ctx, code)
	}
	// デフォルトは存在する（テストケースで明示的に上書きされない限り）
	return true, nil
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
			checker := &mockSymbolChecker{}
			uc := usecase.NewWatchlistUsecase(repo, checker)

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
		name          string
		userID        uint
		symbolCode    string
		checkerExists bool
		checkerErr    error
		repoErr       error
		wantErr       bool
		errIs         error
	}{
		{
			name:          "success: add symbol",
			userID:        1,
			symbolCode:    "TSLA",
			checkerExists: true,
		},
		{
			name:       "failure: empty symbol code",
			userID:     1,
			symbolCode: "",
			wantErr:    true,
		},
		{
			name:          "failure: symbol not in master",
			userID:        1,
			symbolCode:    "UNKNOWN",
			checkerExists: false,
			wantErr:       true,
			errIs:         usecase.ErrSymbolNotInMaster,
		},
		{
			name:       "failure: checker error",
			userID:     1,
			symbolCode: "TSLA",
			checkerErr: errors.New("db error"),
			wantErr:    true,
		},
		{
			name:          "failure: duplicate symbol",
			userID:        1,
			symbolCode:    "AAPL",
			checkerExists: true,
			repoErr:       usecase.ErrSymbolAlreadyExists,
			wantErr:       true,
			errIs:         usecase.ErrSymbolAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var calledUserID uint
			var calledSymbolCode string
			repo := &mockUserSymbolRepository{
				AddWithAtomicSortKeyFunc: func(ctx context.Context, userID uint, symbolCode string) error {
					calledUserID = userID
					calledSymbolCode = symbolCode
					return tt.repoErr
				},
			}
			checker := &mockSymbolChecker{
				ExistsCodeFunc: func(ctx context.Context, code string) (bool, error) {
					return tt.checkerExists, tt.checkerErr
				},
			}
			uc := usecase.NewWatchlistUsecase(repo, checker)

			err := uc.AddSymbol(context.Background(), tt.userID, tt.symbolCode)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errIs != nil {
					assert.ErrorIs(t, err, tt.errIs)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.userID, calledUserID)
				assert.Equal(t, tt.symbolCode, calledSymbolCode)
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
			checker := &mockSymbolChecker{}
			uc := usecase.NewWatchlistUsecase(repo, checker)

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

	currentSymbols := []entity.UserSymbol{
		{UserID: 1, SymbolCode: "AAPL", SortKey: 10},
		{UserID: 1, SymbolCode: "MSFT", SortKey: 20},
		{UserID: 1, SymbolCode: "GOOGL", SortKey: 30},
	}

	tests := []struct {
		name      string
		codeOrder []string
		repoErr   error
		wantErr   bool
		errIs     error
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
			name:      "failure: length mismatch",
			codeOrder: []string{"AAPL"},
			wantErr:   true,
			errIs:     usecase.ErrInvalidReorder,
		},
		{
			name:      "failure: unknown symbol",
			codeOrder: []string{"AAPL", "MSFT", "TSLA"},
			wantErr:   true,
			errIs:     usecase.ErrInvalidReorder,
		},
		{
			name:      "failure: duplicate symbol",
			codeOrder: []string{"AAPL", "AAPL", "GOOGL"},
			wantErr:   true,
			errIs:     usecase.ErrInvalidReorder,
		},
		{
			name:      "failure: repository error",
			codeOrder: []string{"AAPL", "MSFT", "GOOGL"},
			repoErr:   errors.New("database error"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &mockUserSymbolRepository{
				ListByUserFunc: func(ctx context.Context, userID uint) ([]entity.UserSymbol, error) {
					return currentSymbols, nil
				},
				UpdateSortKeysFunc: func(ctx context.Context, userID uint, codeOrder []string) error {
					return tt.repoErr
				},
			}
			checker := &mockSymbolChecker{}
			uc := usecase.NewWatchlistUsecase(repo, checker)

			err := uc.ReorderSymbols(context.Background(), 1, tt.codeOrder)
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
		checker := &mockSymbolChecker{} // デフォルトで全銘柄が存在する
		uc := usecase.NewWatchlistUsecase(repo, checker)

		err := uc.InitializeDefaults(context.Background(), 42)
		require.NoError(t, err)

		assert.Len(t, addedSymbols, len(usecase.DefaultSymbolCodes))
		for i, code := range usecase.DefaultSymbolCodes {
			assert.Equal(t, uint(42), addedSymbols[i].UserID)
			assert.Equal(t, code, addedSymbols[i].SymbolCode)
			assert.Equal(t, (i+1)*10, addedSymbols[i].SortKey)
		}
	})

	t.Run("success: skips symbols not in master", func(t *testing.T) {
		t.Parallel()

		var addedSymbols []entity.UserSymbol
		repo := &mockUserSymbolRepository{
			AddBatchFunc: func(ctx context.Context, userSymbols []entity.UserSymbol) error {
				addedSymbols = userSymbols
				return nil
			},
		}
		// AAPL と GOOGL は存在するが MSFT はマスタに存在しない
		checker := &mockSymbolChecker{
			ExistsCodeFunc: func(ctx context.Context, code string) (bool, error) {
				return code != "MSFT", nil
			},
		}
		uc := usecase.NewWatchlistUsecase(repo, checker)

		err := uc.InitializeDefaults(context.Background(), 42)
		require.NoError(t, err)

		// MSFT がスキップされるため2件
		require.Len(t, addedSymbols, 2)
		assert.Equal(t, "AAPL", addedSymbols[0].SymbolCode)
		assert.Equal(t, "GOOGL", addedSymbols[1].SymbolCode)
	})

	t.Run("failure: repository error", func(t *testing.T) {
		t.Parallel()

		repo := &mockUserSymbolRepository{
			AddBatchFunc: func(ctx context.Context, userSymbols []entity.UserSymbol) error {
				return errors.New("database error")
			},
		}
		checker := &mockSymbolChecker{}
		uc := usecase.NewWatchlistUsecase(repo, checker)

		err := uc.InitializeDefaults(context.Background(), 1)
		assert.Error(t, err)
	})
}
