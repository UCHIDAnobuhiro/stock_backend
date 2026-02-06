// Package adapters はsymbollistフィーチャーのリポジトリ実装を提供します。
package adapters

import (
	"context"

	"stock_backend/internal/feature/symbollist/domain/entity"
	"stock_backend/internal/feature/symbollist/usecase"

	"gorm.io/gorm"
)

// symbolMySQL はSymbolRepositoryインターフェースのMySQL実装です。
type symbolMySQL struct {
	db *gorm.DB
}

var _ usecase.SymbolRepository = (*symbolMySQL)(nil)

// NewSymbolRepository は指定されたDB接続でsymbolMySQLリポジトリの新しいインスタンスを生成します。
func NewSymbolRepository(db *gorm.DB) *symbolMySQL {
	return &symbolMySQL{db: db}
}

// ListActive はsort_key順にすべてのアクティブな銘柄を返します。
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

// ListActiveCodes はsort_key順にアクティブな銘柄のコードのみを返します。
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
