// Package usecase はauthフィーチャーのビジネスロジックを実装します。
package usecase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"stock_backend/internal/feature/auth/domain/entity"
)

const (
	// minPasswordLength はパスワードの最低文字数を定義します。
	minPasswordLength = 8

	// EnvKeyPasswordPepper はパスワードペッパーの環境変数キーです。
	EnvKeyPasswordPepper = "PASSWORD_PEPPER"
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
	pepper       string
	dummyHash    string // タイミング攻撃防止用のダミーハッシュ
}

// NewAuthUsecase はauthUsecaseの新しいインスタンスを生成します。
func NewAuthUsecase(users UserRepository, jwtGenerator JWTGenerator, pepper string) *authUsecase {
	uc := &authUsecase{
		users:        users,
		jwtGenerator: jwtGenerator,
		pepper:       pepper,
	}
	// ペッパー適用済みのダミーハッシュを事前計算（タイミング攻撃防止用）
	pepperedDummy := uc.pepperPassword("dummy")
	dummyHash, _ := bcrypt.GenerateFromPassword([]byte(pepperedDummy), bcrypt.DefaultCost)
	uc.dummyHash = string(dummyHash)
	return uc
}

// pepperPassword はHMAC-SHA256を使用してパスワードにペッパーを適用します。
// bcryptの72バイト制限を回避するため、HMAC-SHA256で固定長のハッシュを生成します。
func (u *authUsecase) pepperPassword(password string) string {
	if u.pepper == "" {
		return password
	}
	mac := hmac.New(sha256.New, []byte(u.pepper))
	mac.Write([]byte(password))
	return hex.EncodeToString(mac.Sum(nil))
}

// validatePassword はパスワードがセキュリティ要件を満たしているかチェックします。
func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters long", minPasswordLength)
	}
	return nil
}

// Signup はハッシュ化されたパスワードで新規ユーザーを登録します。
// 成功時に作成されたユーザーのIDを返します。
func (u *authUsecase) Signup(ctx context.Context, email, password string) (uint, error) {
	// パスワード強度を検証
	if err := validatePassword(password); err != nil {
		return 0, err
	}

	pepperedPassword := u.pepperPassword(password)
	hashed, err := bcrypt.GenerateFromPassword([]byte(pepperedPassword), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}
	user := &entity.User{Email: email, Password: string(hashed)}
	if err := u.users.Create(ctx, user); err != nil {
		return 0, err
	}
	return user.ID, nil
}

// Login はユーザーを認証し、成功時にJWTトークンを返します。
// メールアドレスとパスワードを検証し、署名済みJWTトークンを生成します。
// タイミング攻撃を防止するため、ユーザーが存在しない場合でもbcrypt比較を実行します。
func (u *authUsecase) Login(ctx context.Context, email, password string) (string, error) {
	// メールアドレスでユーザーを検索
	user, err := u.users.FindByEmail(ctx, email)

	// ユーザーが存在しない場合のタイミング攻撃緩和用ダミーハッシュ
	// bcrypt.CompareHashAndPasswordが常に呼ばれることを保証する
	passwordHash := u.dummyHash
	if err == nil {
		passwordHash = user.Password
	}

	// タイミング攻撃防止のため、常にパスワードを検証
	// 第1引数はハッシュ化パスワード、第2引数は平文パスワード
	pepperedPassword := u.pepperPassword(password)
	compareErr := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(pepperedPassword))

	// ユーザー未検出またはパスワード不一致の場合、汎用エラーを返す
	if err != nil || compareErr != nil {
		return "", ErrInvalidCredentials
	}

	// 注入されたジェネレーターを使用してJWTトークンを生成
	token, tokenErr := u.jwtGenerator.GenerateToken(user.ID, user.Email)
	if tokenErr != nil {
		return "", fmt.Errorf("failed to generate token: %w", tokenErr)
	}

	return token, nil
}
