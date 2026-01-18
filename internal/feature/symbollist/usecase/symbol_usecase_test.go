package usecase

import (
	"context"
	"errors"
	"stock_backend/internal/feature/symbollist/domain/entity"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockSymbolRepository is a mock implementation of SymbolRepository interface.
type mockSymbolRepository struct {
	ListActiveFunc      func(ctx context.Context) ([]entity.Symbol, error)
	ListActiveCodesFunc func(ctx context.Context) ([]string, error)
}

func (m *mockSymbolRepository) ListActive(ctx context.Context) ([]entity.Symbol, error) {
	if m.ListActiveFunc != nil {
		return m.ListActiveFunc(ctx)
	}
	return nil, nil
}

func (m *mockSymbolRepository) ListActiveCodes(ctx context.Context) ([]string, error) {
	if m.ListActiveCodesFunc != nil {
		return m.ListActiveCodesFunc(ctx)
	}
	return nil, nil
}

func TestNewSymbolUsecase(t *testing.T) {
	t.Parallel()

	mockRepo := &mockSymbolRepository{}
	uc := NewSymbolUsecase(mockRepo)

	assert.NotNil(t, uc, "usecase should not be nil")
	assert.NotNil(t, uc.repo, "repository should not be nil")
}

func TestSymbolUsecase_ListActiveSymbols(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		mockListActive  func(ctx context.Context) ([]entity.Symbol, error)
		expectedSymbols []entity.Symbol
		wantErr         bool
		errMsg          string
	}{
		{
			name: "success: returns list of active symbols",
			mockListActive: func(ctx context.Context) ([]entity.Symbol, error) {
				return []entity.Symbol{
					{ID: 1, Code: "7203.T", Name: "Toyota Motor", Market: "TSE", IsActive: true, SortKey: 1},
					{ID: 2, Code: "6758.T", Name: "Sony Group", Market: "TSE", IsActive: true, SortKey: 2},
				}, nil
			},
			expectedSymbols: []entity.Symbol{
				{ID: 1, Code: "7203.T", Name: "Toyota Motor", Market: "TSE", IsActive: true, SortKey: 1},
				{ID: 2, Code: "6758.T", Name: "Sony Group", Market: "TSE", IsActive: true, SortKey: 2},
			},
			wantErr: false,
		},
		{
			name: "success: returns empty list when no active symbols",
			mockListActive: func(ctx context.Context) ([]entity.Symbol, error) {
				return []entity.Symbol{}, nil
			},
			expectedSymbols: []entity.Symbol{},
			wantErr:         false,
		},
		{
			name: "success: returns nil when repository returns nil",
			mockListActive: func(ctx context.Context) ([]entity.Symbol, error) {
				return nil, nil
			},
			expectedSymbols: nil,
			wantErr:         false,
		},
		{
			name: "failure: repository returns error",
			mockListActive: func(ctx context.Context) ([]entity.Symbol, error) {
				return nil, errors.New("database connection failed")
			},
			expectedSymbols: nil,
			wantErr:         true,
			errMsg:          "database connection failed",
		},
		{
			name: "success: returns single symbol",
			mockListActive: func(ctx context.Context) ([]entity.Symbol, error) {
				return []entity.Symbol{
					{ID: 1, Code: "9984.T", Name: "SoftBank Group", Market: "TSE", IsActive: true, SortKey: 1},
				}, nil
			},
			expectedSymbols: []entity.Symbol{
				{ID: 1, Code: "9984.T", Name: "SoftBank Group", Market: "TSE", IsActive: true, SortKey: 1},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := &mockSymbolRepository{
				ListActiveFunc: tt.mockListActive,
			}
			uc := NewSymbolUsecase(mockRepo)

			symbols, err := uc.ListActiveSymbols(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.EqualError(t, err, tt.errMsg)
				}
				assert.Nil(t, symbols)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSymbols, symbols)
			}
		})
	}
}

func TestSymbolUsecase_ListActiveSymbols_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	mockRepo := &mockSymbolRepository{
		ListActiveFunc: func(ctx context.Context) ([]entity.Symbol, error) {
			return nil, ctx.Err()
		},
	}
	uc := NewSymbolUsecase(mockRepo)

	symbols, err := uc.ListActiveSymbols(ctx)

	assert.Error(t, err)
	assert.Nil(t, symbols)
	assert.ErrorIs(t, err, context.Canceled)
}
