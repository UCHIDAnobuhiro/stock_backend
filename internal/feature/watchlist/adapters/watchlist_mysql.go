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

// watchlistMySQL はWatchlistRepositoryインターフェースのMySQL実装です。
type watchlistMySQL struct {
	db *gorm.DB
}

var _ usecase.WatchlistRepository = (*watchlistMySQL)(nil)

// NewWatchlistRepository は指定されたDB接続でwatchlistMySQLリポジトリの新しいインスタンスを生成します。
func NewWatchlistRepository(db *gorm.DB) *watchlistMySQL {
	return &watchlistMySQL{db: db}
}

// ListByUser はユーザーのウォッチリストをsort_key昇順で返します。
func (r *watchlistMySQL) ListByUser(ctx context.Context, userID uint) ([]entity.UserSymbol, error) {
	var entries []entity.UserSymbol
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("sort_key ASC").
		Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

// Add はウォッチリストに銘柄を追加します。
// 重複エントリは ErrAlreadyInWatchlist、FK 違反は ErrSymbolNotFound を返します。
func (r *watchlistMySQL) Add(ctx context.Context, entry entity.UserSymbol) error {
	if err := r.db.WithContext(ctx).Create(&entry).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			switch mysqlErr.Number {
			case 1062: // Duplicate entry
				return usecase.ErrAlreadyInWatchlist
			case 1452: // FK constraint violation (symbol not found)
				return usecase.ErrSymbolNotFound
			}
		}
		return err
	}
	return nil
}

// Remove はウォッチリストから銘柄を削除します。
// 対象が存在しない場合は ErrNotInWatchlist を返します。
func (r *watchlistMySQL) Remove(ctx context.Context, userID uint, symbolCode string) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND symbol_code = ?", userID, symbolCode).
		Delete(&entity.UserSymbol{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return usecase.ErrNotInWatchlist
	}
	return nil
}

// UpdateSortKeys はウォッチリストのsort_keyをトランザクション内で一括更新します。
func (r *watchlistMySQL) UpdateSortKeys(ctx context.Context, userID uint, entries []entity.UserSymbol) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, e := range entries {
			if err := tx.Model(&entity.UserSymbol{}).
				Where("user_id = ? AND symbol_code = ?", userID, e.SymbolCode).
				Update("sort_key", e.SortKey).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// MaxSortKey はユーザーのウォッチリストにおける最大のsort_keyを返します。
// ウォッチリストが空の場合は -1 を返します。
func (r *watchlistMySQL) MaxSortKey(ctx context.Context, userID uint) (int, error) {
	var maxKey *int
	if err := r.db.WithContext(ctx).
		Model(&entity.UserSymbol{}).
		Where("user_id = ?", userID).
		Select("MAX(sort_key)").
		Scan(&maxKey).Error; err != nil {
		return 0, err
	}
	if maxKey == nil {
		return -1, nil
	}
	return *maxKey, nil
}
