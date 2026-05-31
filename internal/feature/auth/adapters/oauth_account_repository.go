// Package adapters はauthフィーチャーのリポジトリ実装を提供します。
package adapters

import (
	"context"
	"database/sql"
	"errors"

	"stock_backend/internal/feature/auth/adapters/sqlc"
	"stock_backend/internal/feature/auth/domain/entity"
	"stock_backend/internal/feature/auth/usecase"
)

// oauthAccountRepository は OAuthAccountRepository の sqlc ベース実装です。
type oauthAccountRepository struct {
	q *authsqlc.Queries
}

var _ usecase.OAuthAccountRepository = (*oauthAccountRepository)(nil)

// NewOAuthAccountRepository は指定された *sql.DB で oauthAccountRepository の新しいインスタンスを生成します。
func NewOAuthAccountRepository(db *sql.DB) *oauthAccountRepository {
	return &oauthAccountRepository{q: authsqlc.New(db)}
}

// FindByProvider はプロバイダー名とプロバイダー UID で OAuthAccount を検索します。
// 存在しない場合は usecase.ErrUserNotFound を返します。
func (r *oauthAccountRepository) FindByProvider(ctx context.Context, provider, providerUID string) (*entity.OAuthAccount, error) {
	row, err := r.q.FindOAuthAccountByProvider(ctx, authsqlc.FindOAuthAccountByProviderParams{
		Provider:    provider,
		ProviderUid: providerUID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, usecase.ErrUserNotFound
		}
		return nil, err
	}
	acct := oauthAccountFromSQLC(row)
	return &acct, nil
}

// Create は OAuthAccount を新規作成します。
func (r *oauthAccountRepository) Create(ctx context.Context, acct *entity.OAuthAccount) error {
	if acct == nil {
		return errors.New("account is nil")
	}
	row, err := r.q.CreateOAuthAccount(ctx, authsqlc.CreateOAuthAccountParams{
		UserID:      acct.UserID,
		Provider:    acct.Provider,
		ProviderUid: acct.ProviderUID,
	})
	if err != nil {
		return err
	}
	*acct = oauthAccountFromSQLC(row)
	return nil
}
