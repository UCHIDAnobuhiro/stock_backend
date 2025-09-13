package mysql

import (
	"context"
	"todo_backend/internal/domain/entity"
	"todo_backend/internal/domain/repository"

	"gorm.io/gorm"
)

type symbolMySQL struct {
	db *gorm.DB
}

var _ repository.SymbolRepository = (*symbolMySQL)(nil)

func NewSymbolRepository(db *gorm.DB) repository.SymbolRepository {
	return &symbolMySQL{db: db}
}

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
