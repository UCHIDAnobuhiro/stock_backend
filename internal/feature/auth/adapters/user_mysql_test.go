package adapters

import (
	"context"
	"stock_backend/internal/feature/auth/domain/entity"
	"stock_backend/internal/feature/auth/usecase"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB はテスト用のインメモリSQLiteデータベースを準備します。
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to initialize test database")

	// Userテーブルを作成
	err = db.AutoMigrate(&entity.User{})
	require.NoError(t, err, "failed to migrate table")

	return db
}

// seedUser はテスト用のユーザーをデータベースに作成します。
// このヘルパーはコードの重複を削減し、テストの保守性を向上させます。
func seedUser(t *testing.T, db *gorm.DB, email, password string) *entity.User {
	t.Helper()

	user := &entity.User{
		Email:    email,
		Password: password,
	}
	err := db.Create(user).Error
	require.NoError(t, err, "failed to seed user")

	return user
}

// TestNewUserMySQL はNewUserMySQLコンストラクタが正しくインスタンスを生成することをテストします。
func TestNewUserMySQL(t *testing.T) {
	db := setupTestDB(t)

	repo := NewUserMySQL(db)

	assert.NotNil(t, repo, "repository is nil")
	assert.NotNil(t, repo.db, "database connection is nil")
}

// TestUserMySQL_Create はユーザー作成処理（成功、メール重複、nilユーザー）をテストします。
func TestUserMySQL_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		user         *entity.User
		wantErr      bool
		setupFunc    func(t *testing.T, db *gorm.DB)
		validateFunc func(t *testing.T, user *entity.User)
	}{
		{
			name: "success: user creation",
			user: &entity.User{
				Email:    "test@example.com",
				Password: "hashed_password",
			},
			wantErr: false,
			validateFunc: func(t *testing.T, user *entity.User) {
				assert.NotZero(t, user.ID, "ID is not set")
				assert.False(t, user.CreatedAt.IsZero(), "CreatedAt is not set")
				assert.False(t, user.UpdatedAt.IsZero(), "UpdatedAt is not set")
			},
		},
		{
			name: "failure: duplicate email",
			user: &entity.User{
				Email:    "duplicate@example.com",
				Password: "password2",
			},
			wantErr: true,
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedUser(t, db, "duplicate@example.com", "password1")
			},
		},
		{
			name:    "failure: nil user",
			user:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewUserMySQL(db)

			if tt.setupFunc != nil {
				tt.setupFunc(t, db)
			}

			err := repo.Create(context.Background(), tt.user)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validateFunc != nil {
					tt.validateFunc(t, tt.user)
				}
			}
		})
	}
}

// TestUserMySQL_FindByEmail はメールアドレスによるユーザー検索をテストします。
func TestUserMySQL_FindByEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		email        string
		wantErr      bool
		expectedErr  error
		setupFunc    func(t *testing.T, db *gorm.DB) *entity.User
		validateFunc func(t *testing.T, expected, found *entity.User)
	}{
		{
			name:    "success: find user by email",
			email:   "find@example.com",
			wantErr: false,
			setupFunc: func(t *testing.T, db *gorm.DB) *entity.User {
				return seedUser(t, db, "find@example.com", "hashed_password")
			},
			validateFunc: func(t *testing.T, expected, found *entity.User) {
				assert.NotNil(t, found, "user is nil")
				assert.Equal(t, expected.ID, found.ID, "ID does not match")
				assert.Equal(t, expected.Email, found.Email, "email does not match")
				assert.Equal(t, expected.Password, found.Password, "password does not match")
			},
		},
		{
			name:        "failure: user not found",
			email:       "notfound@example.com",
			wantErr:     true,
			expectedErr: usecase.ErrUserNotFound,
		},
		{
			name:    "failure: empty email",
			email:   "",
			wantErr: true,
		},
		{
			name:    "success: find correct user when multiple users exist",
			email:   "user2@example.com",
			wantErr: false,
			setupFunc: func(t *testing.T, db *gorm.DB) *entity.User {
				seedUser(t, db, "user1@example.com", "pass1")
				user2 := seedUser(t, db, "user2@example.com", "pass2")
				seedUser(t, db, "user3@example.com", "pass3")
				return user2
			},
			validateFunc: func(t *testing.T, expected, found *entity.User) {
				assert.NotNil(t, found, "user is nil")
				assert.Equal(t, expected.ID, found.ID, "ID does not match")
				assert.Equal(t, "user2@example.com", found.Email, "email does not match")
				assert.Equal(t, "pass2", found.Password, "password does not match")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewUserMySQL(db)

			var expected *entity.User
			if tt.setupFunc != nil {
				expected = tt.setupFunc(t, db)
			}

			found, err := repo.FindByEmail(context.Background(), tt.email)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, found, "user should be nil")
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				assert.NoError(t, err)
				if tt.validateFunc != nil {
					tt.validateFunc(t, expected, found)
				}
			}
		})
	}
}

// TestUserMySQL_FindByID はIDによるユーザー検索をテストします。
func TestUserMySQL_FindByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		userID       uint
		wantErr      bool
		expectedErr  error
		setupFunc    func(t *testing.T, db *gorm.DB) *entity.User
		validateFunc func(t *testing.T, expected, found *entity.User)
	}{
		{
			name:    "success: find user by ID",
			wantErr: false,
			setupFunc: func(t *testing.T, db *gorm.DB) *entity.User {
				return seedUser(t, db, "findbyid@example.com", "hashed_password")
			},
			validateFunc: func(t *testing.T, expected, found *entity.User) {
				assert.NotNil(t, found, "user is nil")
				assert.Equal(t, expected.ID, found.ID, "ID does not match")
				assert.Equal(t, expected.Email, found.Email, "email does not match")
				assert.Equal(t, expected.Password, found.Password, "password does not match")
			},
		},
		{
			name:        "failure: user not found",
			userID:      999,
			wantErr:     true,
			expectedErr: usecase.ErrUserNotFound,
		},
		{
			name:    "failure: ID 0",
			userID:  0,
			wantErr: true,
		},
		{
			name:    "success: find correct user when multiple users exist",
			wantErr: false,
			setupFunc: func(t *testing.T, db *gorm.DB) *entity.User {
				seedUser(t, db, "user1@example.com", "pass1")
				user2 := seedUser(t, db, "user2@example.com", "pass2")
				seedUser(t, db, "user3@example.com", "pass3")
				return user2
			},
			validateFunc: func(t *testing.T, expected, found *entity.User) {
				assert.NotNil(t, found, "user is nil")
				assert.Equal(t, expected.ID, found.ID, "ID does not match")
				assert.Equal(t, "user2@example.com", found.Email, "email does not match")
				assert.Equal(t, "pass2", found.Password, "password does not match")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewUserMySQL(db)

			var expected *entity.User
			var targetID uint
			if tt.setupFunc != nil {
				expected = tt.setupFunc(t, db)
				targetID = expected.ID
			} else {
				targetID = tt.userID
			}

			found, err := repo.FindByID(context.Background(), targetID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, found, "user should be nil")
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				assert.NoError(t, err)
				if tt.validateFunc != nil {
					tt.validateFunc(t, expected, found)
				}
			}
		})
	}
}

// TestUserMySQL_Timestamps はCreatedAtとUpdatedAtが自動設定され、取得後も保持されることをテストします。
func TestUserMySQL_Timestamps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "success: CreatedAt and UpdatedAt are automatically set and preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewUserMySQL(db)

			user := &entity.User{
				Email:    "timestamp@example.com",
				Password: "password",
			}

			err := repo.Create(context.Background(), user)
			require.NoError(t, err, "failed to create user")

			assert.False(t, user.CreatedAt.IsZero(), "CreatedAt is not set")
			assert.False(t, user.UpdatedAt.IsZero(), "UpdatedAt is not set")

			// 取得後もタイムスタンプが保持されていることを確認
			found, err := repo.FindByID(context.Background(), user.ID)
			require.NoError(t, err, "failed to find user")

			assert.Equal(t, user.CreatedAt.Unix(), found.CreatedAt.Unix(), "CreatedAt does not match")
			assert.Equal(t, user.UpdatedAt.Unix(), found.UpdatedAt.Unix(), "UpdatedAt does not match")
		})
	}
}
