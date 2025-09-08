package usecase

import (
	"errors"
	"log"
	"os"
	"time"

	"todo_backend/internal/domain"
	"todo_backend/internal/interface/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthUsecaseは認証に関するユースケースを定義するインターフェースです。
// 具体的な実装はインフラ層のDBや外部ライブラリに依存せず、
// ユースケース層からはこの抽象を通して利用されます。
type AuthUsecase interface {
	Signup(email, password string) error
	Login(email, password string) (string, error) // returns JWT
}

// authUsecaseは認証関連のユースケースを表す構造体です。
// UserRepositoryに依存しており、ユーザの作成や取得を行う際に利用する。
type authUsecase struct {
	users repository.UserRepository
}

// NewAuthUsecaseはauthUsecaseの新しいインスタンスを作成する。
// 引数usersには、ユーザの永続化を行うためにUserRepositoryの実装を渡す。
func NewAuthUsecase(users repository.UserRepository) AuthUsecase {
	return &authUsecase{users: users}
}

// SignUpは新規ユーザ登録を行います。
// 受け取ったパスワードはbcryptでハッシュ化し、UserRepository経由で保存します。
// 同じメールアドレスがすでに存在する場合やDBエラーが発生した場合はエラーを返す。
func (u *authUsecase) Signup(email, password string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user := &domain.User{Email: email, Password: string(hashed)}
	return u.users.Create(user)
}

// Loginはユーザ認証を行い、成功した場合はJWTのアクセストークンを返す。
// 1. Emailでユーザ検索
// 2. bcryptでパスワード検証
// 3. JWT_SECRETを使用し、署名つきJWTを生成
// 4. 成功時にログを出力し、トークンを返す
func (u *authUsecase) Login(email, password string) (string, error) {
	// 1. Emailでユーザ検索
	user, err := u.users.FindByEmail(email)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	// 2. bcryptでパスワード検証
	// 第1引数が「ハッシュ」、第2引数が「平文」
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		log.Printf("[LOGIN] bcrypt NG: %v", err)
		return "", errors.New("invalid email or password")
	}
	log.Printf("[LOGIN] bcrypt OK for id=%d", user.ID)

	// 3. JWT_SECRETを使用し、署名つきJWTを生成
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("server misconfigured: JWT_SECRET missing")
	}

	// 4. JWT のクレーム設定
	claims := jwt.MapClaims{
		"sub":   user.ID,                               // ユーザーID（標準: sub）
		"exp":   time.Now().Add(24 * time.Hour).Unix(), // 有効期限（標準: exp）
		"iat":   time.Now().Unix(),                     // 発行時刻（標準: iat）
		"email": user.Email,                            // アプリ独自の公開クレーム
	}

	// 署名付きJWTの生成
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	log.Printf("[LOGIN] success id=%d", user.ID)
	return signed, nil
}
