// Package entity はauthフィーチャーのドメインエンティティを定義します。
package entity

import "time"

// OAuthAccount はOAuth2プロバイダーとユーザーを紐付けるエンティティです。
// (provider, provider_uid) の複合ユニーク制約でプロバイダー側IDの重複を防ぎます。
type OAuthAccount struct {
	ID uint `gorm:"primaryKey"`

	// UserID は紐付けられたユーザーのIDです。
	UserID uint `gorm:"not null;index"`

	// Provider はOAuth2プロバイダー名です（"google" | "github"）。
	Provider string `gorm:"size:32;not null;uniqueIndex:idx_oauth_provider_uid"`

	// ProviderUID はプロバイダー側のユーザー一意IDです。
	// Google: "sub" クレーム / GitHub: ユーザーの数値ID（文字列）
	ProviderUID string `gorm:"size:255;not null;uniqueIndex:idx_oauth_provider_uid"`

	CreatedAt time.Time `gorm:"autoCreateTime;not null"`
}

// TableName はGORMが使用するテーブル名を明示的に指定します。
// 指定しない場合 GORM は OAuthAccount → o_auth_accounts と変換します。
func (OAuthAccount) TableName() string { return "oauth_accounts" }
