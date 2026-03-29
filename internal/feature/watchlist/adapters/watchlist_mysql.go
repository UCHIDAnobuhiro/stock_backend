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
// (user_id, sort_key) のユニーク制約が一時的に違反しないよう、
// まず全レコードを負値（-(i+1)）にシフトしてから最終値に更新します。
func (r *watchlistMySQL) UpdateSortKeys(ctx context.Context, userID uint, entries []entity.UserSymbol) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Phase 1: 負値にシフトして既存の正値との衝突を回避
		for i, e := range entries {
			if err := tx.Model(&entity.UserSymbol{}).
				Where("user_id = ? AND symbol_code = ?", userID, e.SymbolCode).
				Update("sort_key", -(i + 1)).Error; err != nil {
				return err
			}
		}
		// Phase 2: 最終的な sort_key に更新
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

// AddWithNextSortKey はsort_keyをトランザクション内でMAX+1採番して銘柄を追加します。
// SELECT MAX(sort_key) FOR UPDATE と INSERT を同一トランザクションで実行することで、
// 並行追加時の重複順位を防ぎます。
func (r *watchlistMySQL) AddWithNextSortKey(ctx context.Context, userID uint, symbolCode string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maxKey *int
		if err := tx.Model(&entity.UserSymbol{}).
			Where("user_id = ?", userID).
			Select("MAX(sort_key)").
			Set("gorm:query_option", "FOR UPDATE").
			Scan(&maxKey).Error; err != nil {
			return err
		}
		nextKey := 0
		if maxKey != nil {
			nextKey = *maxKey + 1
		}
		entry := entity.UserSymbol{
			UserID:     userID,
			SymbolCode: symbolCode,
			SortKey:    nextKey,
		}
		if err := tx.Create(&entry).Error; err != nil {
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) {
				switch mysqlErr.Number {
				case 1062:
					return usecase.ErrAlreadyInWatchlist
				case 1452:
					return usecase.ErrSymbolNotFound
				}
			}
			return err
		}
		return nil
	})
}
