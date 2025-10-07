package usecase

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"stock_backend/internal/domain/entity"
	"stock_backend/internal/domain/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	jwtExpiration = 1 * time.Hour
)

// AuthUsecase は認証に関するユースケースを定義します。
type AuthUsecase interface {
	Signup(email, password string) error
	Login(email, password string) (string, error) // returns JWT
}

// authUsecase は AuthUsecase の実装です。
type authUsecase struct {
	users repository.UserRepository
}

// NewAuthUsecase は新しい authUsecase を作成します。
func NewAuthUsecase(users repository.UserRepository) AuthUsecase {
	return &authUsecase{users: users}
}

// Signup は新規ユーザーを登録します。
func (u *authUsecase) Signup(email, password string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user := &entity.User{Email: email, Password: string(hashed)}
	return u.users.Create(user)
}

// Login はユーザーを認証し、成功した場合にJWTを返します。
func (u *authUsecase) Login(email, password string) (string, error) {
	// Emailでユーザーを検索
	user, err := u.users.FindByEmail(email)
	if err != nil {
		return "", errors.New("invalid email or password")
	}

	// 2. bcryptでパスワード検証
	// 第1引数が「ハッシュ」、第2引数が「平文」
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		log.Printf("[LOGIN] bcrypt NG: %v", err)
		return "", errors.New("invalid email or password")
	}
	log.Printf("[LOGIN] bcrypt OK for id=%d", user.ID)

	// JWTを生成
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("server misconfigured: JWT_SECRET missing")
	}

	claims := jwt.MapClaims{
		"sub":   user.ID,
		"exp":   time.Now().Add(jwtExpiration).Unix(),
		"iat":   time.Now().Unix(),
		"email": user.Email,
	}

	// 署名
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	log.Printf("[LOGIN] success id=%d", user.ID)
	return signed, nil
}
