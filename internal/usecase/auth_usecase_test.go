package usecase

import (
	"errors"
	"stock_backend/internal/domain/entity"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// mockUserRepository は repository.UserRepository インターフェースのモック実装です。
// テスト中にDBの動作をシミュレートします。
type mockUserRepository struct {
	// CreateFunc を設定すると、Createメソッドが呼ばれたときにこの関数が実行されます。
	CreateFunc func(user *entity.User) error
	// FindByEmailFunc を設定すると、FindByEmailメソッドが呼ばれたときにこの関数が実行されます。
	FindByEmailFunc func(email string) (*entity.User, error)
	// FindByIDFunc を設定すると、FindByIDメソッドが呼ばれたときにこの関数が実行されます。
	FindByIDFunc func(id uint) (*entity.User, error)
}

// Create はモックのCreateメソッドです。
func (m *mockUserRepository) Create(user *entity.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(user)
	}
	return nil // デフォルトでは成功
}

// FindByEmail はモックのFindByEmailメソッドです。
func (m *mockUserRepository) FindByEmail(email string) (*entity.User, error) {
	if m.FindByEmailFunc != nil {
		return m.FindByEmailFunc(email)
	}
	// デフォルトではユーザーが見つからないエラーを返します。
	return nil, errors.New("user not found")
}

// FindByID はモックのFindByIDメソッドです。
func (m *mockUserRepository) FindByID(id uint) (*entity.User, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(id)
	}
	// デフォルトではユーザーが見つからないエラーを返します。
	return nil, errors.New("user not found")
}

func TestAuthUsecase_Signup(t *testing.T) {
	t.Run("サインアップ成功", func(t *testing.T) {
		mockRepo := &mockUserRepository{
			CreateFunc: func(user *entity.User) error {
				// パスワードがハッシュ化されているかを確認
				if len(user.Password) == 0 || user.Password == "password123" {
					t.Errorf("パスワードがハッシュ化されていません")
				}
				// 実際にbcryptとして妥当かも確認
				if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("password123")); err != nil {
					t.Errorf("bcryptハッシュとして無効です: %v", err)
				}
				return nil
			},
		}

		uc := NewAuthUsecase(mockRepo)
		err := uc.Signup("test@example.com", "password123")

		if err != nil {
			t.Errorf("予期せぬエラーが発生しました: %v", err)
		}
	})

	t.Run("リポジトリでの作成失敗", func(t *testing.T) {
		expectedErr := errors.New("database error")
		mockRepo := &mockUserRepository{
			CreateFunc: func(user *entity.User) error {
				return expectedErr
			},
		}

		uc := NewAuthUsecase(mockRepo)
		err := uc.Signup("test@example.com", "password123")

		if !errors.Is(err, expectedErr) {
			t.Errorf("期待したエラー '%v' と異なるエラーが返されました: %v", expectedErr, err)
		}
	})
}

func TestAuthUsecase_Login(t *testing.T) {
	// テスト用のハッシュ化済みパスワード
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	testUser := &entity.User{
		ID:       1,
		Email:    "test@example.com",
		Password: string(hashedPassword),
	}

	// テスト用のJWTシークレットキーを設定
	t.Setenv("JWT_SECRET", "test-secret")

	t.Run("ログイン成功", func(t *testing.T) {
		mockRepo := &mockUserRepository{
			FindByEmailFunc: func(email string) (*entity.User, error) {
				if email == testUser.Email {
					return testUser, nil
				}
				return nil, errors.New("user not found")
			},
		}

		uc := NewAuthUsecase(mockRepo)
		token, err := uc.Login("test@example.com", "password123")

		if err != nil {
			t.Fatalf("予期せぬエラーが発生しました: %v", err)
		}

		if token == "" {
			t.Error("トークンが空です")
		}
	})

	t.Run("ユーザーが見つからない", func(t *testing.T) {
		mockRepo := &mockUserRepository{
			FindByEmailFunc: func(email string) (*entity.User, error) {
				return nil, errors.New("user not found")
			},
		}

		uc := NewAuthUsecase(mockRepo)
		_, err := uc.Login("wrong@example.com", "password123")

		if err == nil {
			t.Fatal("エラーが返されるべきところで、nilが返されました")
		}

		expectedErrMsg := "invalid email or password"
		if err.Error() != expectedErrMsg {
			t.Errorf("期待したエラーメッセージ '%s' と異なるメッセージが返されました: '%s'", expectedErrMsg, err.Error())
		}
	})

	t.Run("パスワードが違う", func(t *testing.T) {
		mockRepo := &mockUserRepository{
			FindByEmailFunc: func(email string) (*entity.User, error) {
				return testUser, nil
			},
		}

		uc := NewAuthUsecase(mockRepo)
		_, err := uc.Login("test@example.com", "wrong-password")

		if err == nil {
			t.Fatal("エラーが返されるべきところで、nilが返されました")
		}

		expectedErrMsg := "invalid email or password"
		if err.Error() != expectedErrMsg {
			t.Errorf("期待したエラーメッセージ '%s' と異なるメッセージが返されました: '%s'", expectedErrMsg, err.Error())
		}
	})

	t.Run("JWTシークレットが設定されていない", func(t *testing.T) {
		// このテストケース内でのみ環境変数を空にする
		t.Setenv("JWT_SECRET", "")

		mockRepo := &mockUserRepository{
			FindByEmailFunc: func(email string) (*entity.User, error) {
				return testUser, nil
			},
		}

		uc := NewAuthUsecase(mockRepo)
		_, err := uc.Login("test@example.com", "password123")

		if err == nil {
			t.Fatal("エラーが返されるべきところで、nilが返されました")
		}

		expectedErrMsg := "server misconfigured: JWT_SECRET missing"
		if err.Error() != expectedErrMsg {
			t.Errorf("期待したエラーメッセージ '%s' と異なるメッセージが返されました: '%s'", expectedErrMsg, err.Error())
		}
	})
}
