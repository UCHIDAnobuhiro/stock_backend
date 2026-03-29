// Package entity はwatchlistフィーチャーのドメインエンティティを定義します。
package entity

import "time"

// UserSymbol はユーザーのウォッチリストエントリを表します。
// watchlists テーブルにマップされ、users.id と symbols.code に FK 制約を持ちます。
type UserSymbol struct {
	ID         uint   `gorm:"primaryKey"`
	UserID     uint   `gorm:"not null;uniqueIndex:idx_watchlist_user_symbol,priority:1"`
	SymbolCode string `gorm:"size:20;not null;uniqueIndex:idx_watchlist_user_symbol,priority:2"`
	SortKey    int    `gorm:"not null;default:0"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
