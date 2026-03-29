// Package entity はwatchlistフィーチャーのドメインエンティティを定義します。
package entity

import "time"

// UserSymbol はユーザーのウォッチリストエントリを表します。
// watchlists テーブルにマップされ、users.id と symbols.code に FK 制約を持ちます。
type UserSymbol struct {
	ID         uint   `gorm:"primaryKey"`
	UserID     uint   `gorm:"not null;uniqueIndex:idx_watchlist_user_symbol,priority:1;uniqueIndex:idx_watchlist_user_sort_key,priority:1"`
	SymbolCode string `gorm:"size:20;not null;uniqueIndex:idx_watchlist_user_symbol,priority:2"`
	SortKey    int    `gorm:"not null;default:0;uniqueIndex:idx_watchlist_user_sort_key,priority:2"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TableName は GORM が使用するテーブル名を返します。
func (UserSymbol) TableName() string {
	return "watchlists"
}
