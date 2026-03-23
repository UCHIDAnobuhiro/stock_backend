// Package adapters はsymbollistフィーチャーのリポジトリ実装を提供します。
package adapters

import (
	"context"

	"gorm.io/gorm"

	"stock_backend/internal/feature/symbollist/domain/entity"
	"stock_backend/internal/feature/symbollist/usecase"
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

// ListActive はコードのアルファベット順にすべてのアクティブな銘柄を返します。
func (r *symbolMySQL) ListActive(ctx context.Context) ([]entity.Symbol, error) {
	var symbols []entity.Symbol
	if err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("code ASC").
		Find(&symbols).Error; err != nil {
		return nil, err
	}
	return symbols, nil
}

// ListActiveCodes はアクティブな銘柄のコードのみを返します。
func (r *symbolMySQL) ListActiveCodes(ctx context.Context) ([]string, error) {
	var codes []string
	if err := r.db.WithContext(ctx).
		Model(&entity.Symbol{}).
		Where("is_active = ?", true).
		Pluck("code", &codes).Error; err != nil {
		return nil, err
	}
	return codes, nil
}

// ExistsCode は指定されたコードがアクティブなマスタ銘柄として存在するかを確認します。
// watchlist の SymbolChecker インターフェースを DI 経由で満たします。
func (r *symbolMySQL) ExistsCode(ctx context.Context, code string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.Symbol{}).
		Where("code = ? AND is_active = ?", code, true).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
