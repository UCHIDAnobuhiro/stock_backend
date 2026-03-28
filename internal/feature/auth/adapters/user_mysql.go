// Package adapters はauthフィーチャーのリポジトリ実装を提供します。
package adapters

import (
	"context"
	"errors"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"stock_backend/internal/feature/auth/domain/entity"
	"stock_backend/internal/feature/auth/usecase"
)

// userMySQL はUserRepositoryインターフェースのMySQL実装です。
// GORMを使用してデータベース操作を行います。
type userMySQL struct {
	db *gorm.DB
}

// userMySQLがUserRepositoryを実装していることをコンパイル時に検証します。
var _ usecase.UserRepository = (*userMySQL)(nil)

// NewUserMySQL は指定されたgorm.DB接続でuserMySQLの新しいインスタンスを生成します。
// 依存性注入用のコンストラクタです。
func NewUserMySQL(db *gorm.DB) *userMySQL {
	return &userMySQL{db: db}
}

// Create はユーザーをデータベースに追加します。
// 同じメールアドレスのユーザーが既に存在する場合、usecase.ErrEmailAlreadyExistsを返します。
func (r *userMySQL) Create(ctx context.Context, u *entity.User) error {
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		// MySQLエラー1062: ユニークキーの重複エントリ
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return usecase.ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}

// FindByEmail はメールアドレスでユーザーを取得します。
// ユーザーが存在しない場合、usecase.ErrUserNotFoundを返します。
func (r *userMySQL) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var u entity.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, usecase.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// FindByID はIDでユーザーを取得します。
// ユーザーが存在しない場合、usecase.ErrUserNotFoundを返します。
func (r *userMySQL) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	var u entity.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, usecase.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}
