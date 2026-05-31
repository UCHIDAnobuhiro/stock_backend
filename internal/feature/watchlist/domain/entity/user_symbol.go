// Package entity はwatchlistフィーチャーのドメインエンティティを定義します。
package entity

import "time"

// UserSymbol はユーザーのウォッチリストエントリを表します。
// watchlists テーブルにマップされ、users.id と symbols.code に FK 制約を持ちます。
type UserSymbol struct {
	ID         int64
	UserID     int64
	SymbolCode string
	SortKey    int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
