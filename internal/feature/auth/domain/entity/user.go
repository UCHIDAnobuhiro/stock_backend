// Package entity はauthフィーチャーのドメインエンティティを定義します。
package entity

import "time"

// User はシステムに登録されたユーザーを表します。
// 認証情報とユーザー管理用のメタデータを含みます。
type User struct {
	// ID はユーザーの一意な識別子です。
	ID uint `gorm:"primaryKey"`

	// Email は認証に使用されるユーザーのメールアドレスです。
	// 全ユーザー間で一意である必要があります。
	Email string `gorm:"uniqueIndex;size:255;not null"`

	// Password はユーザーのハッシュ化されたパスワードです。
	// 平文パスワードを保存してはなりません。
	Password string `gorm:"size:255;not null"`

	// CreatedAt はユーザーが作成された日時です。
	CreatedAt time.Time

	// UpdatedAt はユーザーが最後に更新された日時です。
	UpdatedAt time.Time
}
