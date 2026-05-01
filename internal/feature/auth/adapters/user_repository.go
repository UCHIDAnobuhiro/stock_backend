// Package adapters はauthフィーチャーのリポジトリ実装を提供します。
package adapters

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"stock_backend/internal/feature/auth/domain/entity"
	"stock_backend/internal/feature/auth/usecase"
)

// userRepository はUserRepositoryインターフェースのリポジトリ実装です。
// GORMを使用してデータベース操作を行います。
type userRepository struct {
	db *gorm.DB
}

// userRepositoryがUserRepositoryおよびOAuthUserCreatorを実装していることをコンパイル時に検証します。
var _ usecase.UserRepository = (*userRepository)(nil)
var _ usecase.OAuthUserCreator = (*userRepository)(nil)

// NewUserRepository は指定されたgorm.DB接続でuserRepositoryの新しいインスタンスを生成します。
// 依存性注入用のコンストラクタです。
func NewUserRepository(db *gorm.DB) *userRepository {
	return &userRepository{db: db}
}

// Create はユーザーをデータベースに追加します。
// 同じメールアドレスのユーザーが既に存在する場合、usecase.ErrEmailAlreadyExistsを返します。
func (r *userRepository) Create(ctx context.Context, u *entity.User) error {
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		// PostgreSQLエラー23505: ユニーク制約違反
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return usecase.ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}

// FindByEmail はメールアドレスでユーザーを取得します。
// ユーザーが存在しない場合、usecase.ErrUserNotFoundを返します。
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var u entity.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, usecase.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// CreateUserWithOAuthAccount はUserとOAuthAccountをトランザクション内で原子的に作成します。
// OAuthUserCreatorインターフェースの実装です。
func (r *userRepository) CreateUserWithOAuthAccount(ctx context.Context, user *entity.User, account *entity.OAuthAccount) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return usecase.ErrEmailAlreadyExists
			}
			return err
		}
		account.UserID = user.ID
		return tx.Create(account).Error
	})
}

// FindByID はIDでユーザーを取得します。
// ユーザーが存在しない場合、usecase.ErrUserNotFoundを返します。
func (r *userRepository) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	var u entity.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, usecase.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}
