// Package entity はsymbollistフィーチャーのドメインモデルを定義します。
package entity

import "time"

// Symbol はシステム内の株式銘柄コードを表します。
// 銘柄コード、企業名、市場などの取引証券に関する情報を保持します。
type Symbol struct {
	ID            uint       `gorm:"primaryKey"`                   // 主キー
	Code          string     `gorm:"size:20;not null;uniqueIndex"` // 銘柄コード（例: "AAPL", "7203.T"）
	Name          string     `gorm:"size:255;not null"`            // 企業名
	Market        string     `gorm:"size:100;not null"`            // 市場識別子（例: "NASDAQ", "TSE"）
	Timezone      string     `gorm:"size:64;not null"`             // 取引所の IANA タイムゾーン（例: "America/New_York", "Asia/Tokyo"）
	LogoURL       *string    `gorm:"type:text"`                    // Twelve DataのロゴURL（未取得時はNULL）
	LogoUpdatedAt *time.Time // ロゴURLを最後に取得・更新した日時
	IsActive      bool       `gorm:"not null;default:true"`   // トラッキング対象かどうか
	CreatedAt     time.Time  `gorm:"autoCreateTime;not null"` // 登録日時
	UpdatedAt     time.Time  `gorm:"autoUpdateTime;not null"` // 最終更新日時
}
