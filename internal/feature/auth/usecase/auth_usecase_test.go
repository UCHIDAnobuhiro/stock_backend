package usecase_test

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"stock_backend/internal/feature/auth/domain/entity"
	"stock_backend/internal/feature/auth/usecase"
)

// mockUserRepository はUserRepositoryインターフェースのモック実装です。
// テスト中のデータベース操作をシミュレートします。
type mockUserRepository struct {
	// CreateFunc はCreateメソッド呼び出し時に実行されます。
	CreateFunc func(ctx context.Context, user *entity.User) error
	// FindByEmailFunc はFindByEmailメソッド呼び出し時に実行されます。
	FindByEmailFunc func(ctx context.Context, email string) (*entity.User, error)
	// FindByIDFunc はFindByIDメソッド呼び出し時に実行されます。
	FindByIDFunc func(ctx context.Context, id uint) (*entity.User, error)
}

// mockJWTGenerator はJWTGeneratorインターフェースのモック実装です。
// テスト中のJWTトークン生成をシミュレートします。
type mockJWTGenerator struct {
	// GenerateTokenFunc はGenerateTokenメソッド呼び出し時に実行されます。
	GenerateTokenFunc func(userID uint, email string) (string, error)
}

// GenerateToken はGenerateTokenメソッドのモック実装です。
func (m *mockJWTGenerator) GenerateToken(userID uint, email string) (string, error) {
	if m.GenerateTokenFunc != nil {
		return m.GenerateTokenFunc(userID, email)
	}
	// デフォルト: ダミートークンを返す
	return "mock-jwt-token", nil
}

// Create はCreateメソッドのモック実装です。
func (m *mockUserRepository) Create(ctx context.Context, user *entity.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, user)
	}
	return nil // デフォルト: 成功
}

// FindByEmail はFindByEmailメソッドのモック実装です。
func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	if m.FindByEmailFunc != nil {
		return m.FindByEmailFunc(ctx, email)
	}
	// デフォルト: ユーザー未検出エラーを返す
	return nil, errors.New("user not found")
}

// FindByID はFindByIDメソッドのモック実装です。
func (m *mockUserRepository) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	// デフォルト: ユーザー未検出エラーを返す
	return nil, errors.New("user not found")
}

// createTestUser はテスト用にハッシュ化パスワードを持つテストユーザーを作成します。
// このヘルパーはコードの重複を削減し、テストの保守性を向上させます。
func createTestUser(t *testing.T, id uint, email, password string) *entity.User {
	t.Helper()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return &entity.User{
		ID:       id,
		Email:    email,
		Password: string(hashedPassword),
	}
}

// assertError はエラーが期待値と一致するかチェックします。
// このヘルパーは全テストのエラーアサーションを標準化します。
func assertError(t *testing.T, err error, wantErr bool, errMsg string) {
	t.Helper()
	if wantErr {
		if err == nil {
			t.Fatal("expected error but got nil")
		}
		if errMsg != "" && err.Error() != errMsg {
			t.Errorf("expected error '%s', got: '%s'", errMsg, err.Error())
		}
	} else {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

// verifyBcryptHash はハッシュ化パスワードが平文パスワードと一致するかチェックします。
// このヘルパーはbcrypt検証ロジックをカプセル化します。
func verifyBcryptHash(t *testing.T, hashedPassword, plainPassword string) {
	t.Helper()
	if len(hashedPassword) == 0 || hashedPassword == plainPassword {
		t.Error("password is not hashed")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword)); err != nil {
		t.Errorf("invalid bcrypt hash: %v", err)
	}
}

// TestAuthUsecase_Signup はサインアップのビジネスロジック（パスワード検証、ハッシュ化、リポジトリ呼び出し）をテストします。
func TestAuthUsecase_Signup(t *testing.T) {
	t.Parallel() // テスト関数の並列実行を有効化

	tests := []struct {
		name             string
		email            string
		password         string
		wantErr          bool
		errMsg           string
		verifyBcryptHash bool
		repositoryErr    error
	}{
		{
			name:             "successful signup",
			email:            "test@example.com",
			password:         "password123",
			wantErr:          false,
			verifyBcryptHash: true,
		},
		{
			name:     "password too short",
			email:    "test@example.com",
			password: "short",
			wantErr:  true,
			errMsg:   "password must be at least 8 characters long",
		},
		{
			name:             "password at minimum length",
			email:            "test@example.com",
			password:         "12345678",
			wantErr:          false,
			verifyBcryptHash: true,
		},
		{
			name:     "empty password",
			email:    "test@example.com",
			password: "",
			wantErr:  true,
			errMsg:   "password must be at least 8 characters long",
		},
		{
			name:             "long password",
			email:            "test@example.com",
			password:         "this-is-a-very-long-password-with-many-characters",
			wantErr:          false,
			verifyBcryptHash: true,
		},
		{
			name:          "repository create failure",
			email:         "test@example.com",
			password:      "password123",
			wantErr:       true,
			repositoryErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // サブテストの並列実行を有効化

			mockRepo := &mockUserRepository{
				CreateFunc: func(ctx context.Context, user *entity.User) error {
					if tt.verifyBcryptHash {
						verifyBcryptHash(t, user.Password, tt.password)
					}
					if tt.repositoryErr != nil {
						return tt.repositoryErr
					}
					return nil
				},
			}
			mockJWT := &mockJWTGenerator{}

			uc := usecase.NewAuthUsecase(mockRepo, mockJWT)
			err := uc.Signup(context.Background(), tt.email, tt.password)

			// Assert error expectations
			assertError(t, err, tt.wantErr, tt.errMsg)
			if tt.repositoryErr != nil && !errors.Is(err, tt.repositoryErr) {
				t.Errorf("expected error '%v', got: %v", tt.repositoryErr, err)
			}
		})
	}
}

func TestAuthUsecase_Login(t *testing.T) {
	t.Parallel() // enable parallel execution for test function

	// Create test user using helper function
	testUser := createTestUser(t, 1, "test@example.com", "password123")

	tests := []struct {
		name              string
		email             string
		password          string
		wantErr           bool
		errMsg            string
		expectedToken     string
		findByEmailResult *entity.User
		findByEmailErr    error
		jwtGenerateErr    error
		verifyJWTParams   bool
	}{
		{
			name:              "successful login",
			email:             "test@example.com",
			password:          "password123",
			wantErr:           false,
			expectedToken:     "mock-jwt-token",
			findByEmailResult: testUser,
			verifyJWTParams:   true,
		},
		{
			name:           "user not found",
			email:          "wrong@example.com",
			password:       "password123",
			wantErr:        true,
			errMsg:         "invalid email or password",
			findByEmailErr: errors.New("user not found"),
		},
		{
			name:              "incorrect password",
			email:             "test@example.com",
			password:          "wrong-password",
			wantErr:           true,
			errMsg:            "invalid email or password",
			findByEmailResult: testUser,
		},
		{
			name:              "JWT generation failure",
			email:             "test@example.com",
			password:          "password123",
			wantErr:           true,
			errMsg:            "failed to generate token: failed to sign token",
			findByEmailResult: testUser,
			jwtGenerateErr:    errors.New("failed to sign token"),
		},
		{
			name:              "edge case: empty password with valid user",
			email:             "test@example.com",
			password:          "",
			wantErr:           true,
			errMsg:            "invalid email or password",
			findByEmailResult: testUser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // サブテストの並列実行を有効化

			mockRepo := &mockUserRepository{
				FindByEmailFunc: func(ctx context.Context, email string) (*entity.User, error) {
					if tt.findByEmailErr != nil {
						return nil, tt.findByEmailErr
					}
					return tt.findByEmailResult, nil
				},
			}
			mockJWT := &mockJWTGenerator{
				GenerateTokenFunc: func(userID uint, email string) (string, error) {
					if tt.verifyJWTParams {
						if userID != testUser.ID || email != testUser.Email {
							t.Errorf("unexpected userID or email: got userID=%d, email=%s", userID, email)
						}
					}
					if tt.jwtGenerateErr != nil {
						return "", tt.jwtGenerateErr
					}
					return tt.expectedToken, nil
				},
			}

			uc := usecase.NewAuthUsecase(mockRepo, mockJWT)
			token, err := uc.Login(context.Background(), tt.email, tt.password)

			// エラーの期待値を検証
			assertError(t, err, tt.wantErr, tt.errMsg)

			// 成功ケースの期待値を検証
			if !tt.wantErr {
				if token == "" {
					t.Error("token is empty")
				}
				if tt.expectedToken != "" && token != tt.expectedToken {
					t.Errorf("expected token '%s', got: '%s'", tt.expectedToken, token)
				}
			}
		})
	}
}
