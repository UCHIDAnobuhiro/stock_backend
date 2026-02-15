// Package entity はsymbollistフィーチャーのドメインモデルを定義します。
package entity

import "time"

// Symbol はシステム内の株式銘柄コードを表します。
// 銘柄コード、企業名、市場、表示順序などの取引証券に関する情報を保持します。
type Symbol struct {
	ID        uint      `gorm:"primaryKey"`                   // 主キー
	Code      string    `gorm:"size:20;not null;uniqueIndex"` // 銘柄コード（例: "AAPL", "7203.T"）
	Name      string    `gorm:"size:255;not null"`            // 企業名
	Market    string    `gorm:"size:100;not null"`            // 市場識別子（例: "NASDAQ", "TSE"）
	IsActive  bool      `gorm:"not null;default:true"`        // トラッキング対象かどうか
	SortKey   int       `gorm:"not null;default:0"`           // 表示順序の優先度
	UpdatedAt time.Time `gorm:"autoUpdateTime"`               // 最終更新日時
}
