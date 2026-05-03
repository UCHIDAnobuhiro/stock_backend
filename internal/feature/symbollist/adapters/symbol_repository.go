// Package adapters はsymbollistフィーチャーのリポジトリ実装を提供します。
package adapters

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"stock_backend/internal/feature/symbollist/domain/entity"
	"stock_backend/internal/feature/symbollist/usecase"
)

// symbolRepository はSymbolRepositoryインターフェースのリポジトリ実装です。
type symbolRepository struct {
	db *gorm.DB
}

var (
	_ usecase.SymbolRepository     = (*symbolRepository)(nil)
	_ usecase.LogoSymbolRepository = (*symbolRepository)(nil)
)

// NewSymbolRepository は指定されたDB接続でsymbolRepositoryの新しいインスタンスを生成します。
func NewSymbolRepository(db *gorm.DB) *symbolRepository {
	return &symbolRepository{db: db}
}

// ListActive はコード昇順にすべてのアクティブな銘柄を返します。
func (r *symbolRepository) ListActive(ctx context.Context) ([]entity.Symbol, error) {
	var symbols []entity.Symbol
	if err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("code ASC").
		Find(&symbols).Error; err != nil {
		return nil, err
	}
	return symbols, nil
}

// UpdateLogoURL は指定された銘柄のロゴURLと取得日時を更新します。
// 対象行が存在しない場合はエラーとせず警告ログを出力します（バッチの続行を優先するため）。
func (r *symbolRepository) UpdateLogoURL(ctx context.Context, code, logoURL string, updatedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&entity.Symbol{}).
		Where("code = ?", code).
		Updates(map[string]any{
			"logo_url":        logoURL,
			"logo_updated_at": updatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		slog.Warn("UpdateLogoURL: no matching symbol found", "code", code)
	}
	return nil
}

// ListActiveCodes はコード昇順にアクティブな銘柄のコードのみを返します。
func (r *symbolRepository) ListActiveCodes(ctx context.Context) ([]string, error) {
	var codes []string
	if err := r.db.WithContext(ctx).
		Model(&entity.Symbol{}).
		Where("is_active = ?", true).
		Order("code ASC").
		Pluck("code", &codes).Error; err != nil {
		return nil, err
	}
	return codes, nil
}

// Exists は指定されたコードの銘柄が存在するかを返します。
func (r *symbolRepository) Exists(ctx context.Context, code string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&entity.Symbol{}).
		Where("code = ?", code).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
