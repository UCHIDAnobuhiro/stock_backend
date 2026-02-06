// Package usecase はauthフィーチャーのビジネスロジックを実装します。
package usecase

import (
	"context"
	"errors"
	"fmt"

	"stock_backend/internal/feature/auth/domain/entity"

	"golang.org/x/crypto/bcrypt"
)

const (
	// minPasswordLength はパスワードの最低文字数を定義します。
	minPasswordLength = 8
)

// UserRepository はユーザーエンティティの永続化層を抽象化します。
// Goの慣例に従い、インターフェースはプロバイダー（adapters）ではなくコンシューマー（usecase）が定義します。
type UserRepository interface {
	// Create は新しいユーザーをストレージに永続化します。
	// 同じメールアドレスのユーザーが既に存在する場合、エラーを返します。
	Create(ctx context.Context, user *entity.User) error

	// FindByEmail は指定されたメールアドレスに一致するユーザーを取得します。
	// ユーザーが存在しない場合、エラーを返します。
	FindByEmail(ctx context.Context, email string) (*entity.User, error)

	// FindByID は指定されたIDに一致するユーザーを取得します。
	// ユーザーが存在しない場合、エラーを返します。
	FindByID(ctx context.Context, id uint) (*entity.User, error)
}

// JWTGenerator はJWTトークン生成のインターフェースを定義します。
// Goの慣例に従い、インターフェースはプロバイダー（platform/jwt）ではなくコンシューマー（usecase）が定義します。
type JWTGenerator interface {
	// GenerateToken は指定されたユーザーの署名済みJWTトークンを生成します。
	GenerateToken(userID uint, email string) (string, error)
}

// authUsecase は認証ビジネスロジックを実装します。
type authUsecase struct {
	users        UserRepository
	jwtGenerator JWTGenerator
}

// NewAuthUsecase はauthUsecaseの新しいインスタンスを生成します。
func NewAuthUsecase(users UserRepository, jwtGenerator JWTGenerator) *authUsecase {
	return &authUsecase{
		users:        users,
		jwtGenerator: jwtGenerator,
	}
}

// validatePassword はパスワードがセキュリティ要件を満たしているかチェックします。
func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters long", minPasswordLength)
	}
	return nil
}

// Signup はハッシュ化されたパスワードで新規ユーザーを登録します。
func (u *authUsecase) Signup(ctx context.Context, email, password string) error {
	// パスワード強度を検証
	if err := validatePassword(password); err != nil {
		return err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user := &entity.User{Email: email, Password: string(hashed)}
	return u.users.Create(ctx, user)
}

// Login はユーザーを認証し、成功時にJWTトークンを返します。
// メールアドレスとパスワードを検証し、署名済みJWTトークンを生成します。
// タイミング攻撃を防止するため、ユーザーが存在しない場合でもbcrypt比較を実行します。
func (u *authUsecase) Login(ctx context.Context, email, password string) (string, error) {
	// メールアドレスでユーザーを検索
	user, err := u.users.FindByEmail(ctx, email)

	// ユーザーが存在しない場合のタイミング攻撃緩和用ダミーハッシュ
	// bcrypt.CompareHashAndPasswordが常に呼ばれることを保証する
	passwordHash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy" // ダミーハッシュ
	if err == nil {
		passwordHash = user.Password
	}

	// タイミング攻撃防止のため、常にパスワードを検証
	// 第1引数はハッシュ化パスワード、第2引数は平文パスワード
	compareErr := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))

	// ユーザー未検出またはパスワード不一致の場合、汎用エラーを返す
	if err != nil || compareErr != nil {
		return "", errors.New("invalid email or password")
	}

	// 注入されたジェネレーターを使用してJWTトークンを生成
	token, tokenErr := u.jwtGenerator.GenerateToken(user.ID, user.Email)
	if tokenErr != nil {
		return "", fmt.Errorf("failed to generate token: %w", tokenErr)
	}

	return token, nil
}
