// Package adapters はwatchlistフィーチャーのリポジトリ実装を提供します。
package adapters

import (
	"context"
	"errors"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"stock_backend/internal/feature/watchlist/domain/entity"
	"stock_backend/internal/feature/watchlist/usecase"
)

// userSymbolMySQL はUserSymbolRepositoryインターフェースのMySQL実装です。
type userSymbolMySQL struct {
	db *gorm.DB
}

// userSymbolMySQLがUserSymbolRepositoryを実装していることをコンパイル時に検証します。
var _ usecase.UserSymbolRepository = (*userSymbolMySQL)(nil)

// NewUserSymbolRepository は指定されたgorm.DB接続でuserSymbolMySQLの新しいインスタンスを生成します。
func NewUserSymbolRepository(db *gorm.DB) *userSymbolMySQL {
	return &userSymbolMySQL{db: db}
}

// ListByUser はユーザーのウォッチリスト銘柄をsort_key順に返します。
func (r *userSymbolMySQL) ListByUser(ctx context.Context, userID uint) ([]entity.UserSymbol, error) {
	var symbols []entity.UserSymbol
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("sort_key ASC").
		Find(&symbols).Error; err != nil {
		return nil, err
	}
	return symbols, nil
}

// Add はユーザーのウォッチリストに銘柄を追加します。
// 既に同じ銘柄が存在する場合、ErrSymbolAlreadyExistsを返します。
func (r *userSymbolMySQL) Add(ctx context.Context, us *entity.UserSymbol) error {
	if err := r.db.WithContext(ctx).Create(us).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return usecase.ErrSymbolAlreadyExists
		}
		return err
	}
	return nil
}

// Remove はユーザーのウォッチリストから銘柄を削除します。
// 銘柄が存在しない場合、ErrSymbolNotFoundを返します。
func (r *userSymbolMySQL) Remove(ctx context.Context, userID uint, symbolCode string) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND symbol_code = ?", userID, symbolCode).
		Delete(&entity.UserSymbol{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return usecase.ErrSymbolNotFound
	}
	return nil
}

// UpdateSortKeys はユーザーの銘柄の並び順を一括更新します。
// codeOrderの順番に従い、sort_key = index * 10 で設定します。
func (r *userSymbolMySQL) UpdateSortKeys(ctx context.Context, userID uint, codeOrder []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, code := range codeOrder {
			result := tx.Model(&entity.UserSymbol{}).
				Where("user_id = ? AND symbol_code = ?", userID, code).
				Update("sort_key", i*10)
			if result.Error != nil {
				return result.Error
			}
		}
		return nil
	})
}

// AddBatch はユーザーのウォッチリストに複数の銘柄を一括追加します。
// 既に存在する銘柄は無視されます（冪等性を保証）。
func (r *userSymbolMySQL) AddBatch(ctx context.Context, userSymbols []entity.UserSymbol) error {
	if len(userSymbols) == 0 {
		return nil
	}
	// Clauses(clause.Insert{Modifier: "IGNORE"}) の代わりに個別挿入で冪等性を保証
	for i := range userSymbols {
		if err := r.db.WithContext(ctx).Create(&userSymbols[i]).Error; err != nil {
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
				continue // 重複は無視
			}
			return err
		}
	}
	return nil
}

// MaxSortKey はユーザーのウォッチリスト内の最大sort_keyを返します。
// 銘柄が存在しない場合は0を返します。
func (r *userSymbolMySQL) MaxSortKey(ctx context.Context, userID uint) (int, error) {
	var maxKey *int
	err := r.db.WithContext(ctx).
		Model(&entity.UserSymbol{}).
		Where("user_id = ?", userID).
		Select("MAX(sort_key)").
		Scan(&maxKey).Error
	if err != nil {
		return 0, err
	}
	if maxKey == nil {
		return 0, nil
	}
	return *maxKey, nil
}
