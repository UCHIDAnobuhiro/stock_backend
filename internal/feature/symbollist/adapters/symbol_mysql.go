// Package adapters provides repository implementations for the symbollist feature.
package adapters

import (
	"context"

	"stock_backend/internal/feature/symbollist/domain/entity"
	"stock_backend/internal/feature/symbollist/usecase"

	"gorm.io/gorm"
)

// symbolMySQL is a MySQL implementation of the SymbolRepository interface.
type symbolMySQL struct {
	db *gorm.DB
}

var _ usecase.SymbolRepository = (*symbolMySQL)(nil)

// NewSymbolRepository creates a new symbolMySQL repository with the given database connection.
func NewSymbolRepository(db *gorm.DB) *symbolMySQL {
	return &symbolMySQL{db: db}
}

// ListActive returns all active symbols ordered by sort_key.
func (r *symbolMySQL) ListActive(ctx context.Context) ([]entity.Symbol, error) {
	var symbols []entity.Symbol
	if err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("sort_key ASC").
		Find(&symbols).Error; err != nil {
		return nil, err
	}
	return symbols, nil
}

// ListActiveCodes returns only the codes of active symbols ordered by sort_key.
func (r *symbolMySQL) ListActiveCodes(ctx context.Context) ([]string, error) {
	var codes []string
	if err := r.db.WithContext(ctx).
		Model(&entity.Symbol{}).
		Where("is_active = ?", true).
		Order("sort_key ASC").
		Pluck("code", &codes).Error; err != nil {
		return nil, err
	}
	return codes, nil
}
