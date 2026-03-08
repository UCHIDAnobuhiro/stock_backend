// Package entity はwatchlistフィーチャーのドメインモデルを定義します。
package entity

import "time"

// UserSymbol はユーザー固有のウォッチリスト銘柄を表します。
type UserSymbol struct {
	ID         uint   `gorm:"primaryKey"`
	UserID     uint   `gorm:"not null;uniqueIndex:uk_user_symbol"`
	SymbolCode string `gorm:"size:20;not null;uniqueIndex:uk_user_symbol;column:symbol_code"`
	SortKey    int    `gorm:"not null;default:0"`
	CreatedAt  time.Time
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}
