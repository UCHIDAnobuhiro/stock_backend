// Package adapters はauthフィーチャーのリポジトリ実装を提供します。
package adapters

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"stock_backend/internal/feature/auth/domain/entity"
	"stock_backend/internal/feature/auth/usecase"
)

// oauthAccountRepository はOAuthAccountRepositoryインターフェースのGORM実装です。
type oauthAccountRepository struct {
	db *gorm.DB
}

var _ usecase.OAuthAccountRepository = (*oauthAccountRepository)(nil)

// NewOAuthAccountRepository は指定されたgorm.DB接続でoauthAccountRepositoryの新しいインスタンスを生成します。
func NewOAuthAccountRepository(db *gorm.DB) *oauthAccountRepository {
	return &oauthAccountRepository{db: db}
}

// FindByProvider はプロバイダー名とプロバイダーUIDでOAuthAccountを検索します。
// 存在しない場合はusecase.ErrUserNotFoundを返します。
func (r *oauthAccountRepository) FindByProvider(ctx context.Context, provider, providerUID string) (*entity.OAuthAccount, error) {
	var acct entity.OAuthAccount
	err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_uid = ?", provider, providerUID).
		First(&acct).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, usecase.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &acct, nil
}

// Create はOAuthAccountを新規作成します。
func (r *oauthAccountRepository) Create(ctx context.Context, acct *entity.OAuthAccount) error {
	return r.db.WithContext(ctx).Create(acct).Error
}
