// Package adapters はauthフィーチャーのリポジトリ実装を提供します。
package adapters

import (
	"log/slog"

	"gorm.io/gorm"

	authentity "stock_backend/internal/feature/auth/domain/entity"
)

// MakePasswordNullable はusersテーブルのpasswordカラムをNULL許容に変更します。
// GORMのAutoMigrateはNOT NULL制約の削除に対応しないため、DDLを直接実行します。
// information_schemaで現在の状態を確認してから変更するため冪等です。
func MakePasswordNullable(db *gorm.DB) error {
	var count int64
	err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_name = 'users'
		  AND column_name = 'password'
		  AND is_nullable = 'NO'
	`).Scan(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		slog.Info("users.password is already nullable, skipping")
		return nil
	}
	if err := db.Exec(`ALTER TABLE users ALTER COLUMN password DROP NOT NULL`).Error; err != nil {
		return err
	}
	slog.Info("altered users.password to nullable")
	return nil
}

// AddOAuthAccountsFKConstraints はoauth_accountsテーブルのFK制約を冪等に追加します。
func AddOAuthAccountsFKConstraints(db *gorm.DB) error {
	if !db.Migrator().HasConstraint(&authentity.OAuthAccount{}, "fk_oauth_accounts_user") {
		if err := db.Exec(`ALTER TABLE oauth_accounts ADD CONSTRAINT fk_oauth_accounts_user
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`).Error; err != nil {
			return err
		}
		slog.Info("added FK constraint: fk_oauth_accounts_user")
	}
	return nil
}
