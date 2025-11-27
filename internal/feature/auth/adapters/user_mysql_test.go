package adapters

import (
	"stock_backend/internal/feature/auth/domain/entity"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBはテスト用のin-memory SQLiteデータベースを準備します。
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "テスト用DBの初期化に失敗しました")

	// Userテーブルを作成
	err = db.AutoMigrate(&entity.User{})
	require.NoError(t, err, "テーブルのマイグレーションに失敗しました")

	return db
}

func TestNewUserMySQL(t *testing.T) {
	db := setupTestDB(t)

	repo := NewUserMySQL(db)

	assert.NotNil(t, repo, "リポジトリがnilです")
	assert.NotNil(t, repo.db, "DBコネクションがnilです")
}

func TestUserMySQL_Create(t *testing.T) {
	t.Run("ユーザー作成成功", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		user := &entity.User{
			Email:    "test@example.com",
			Password: "hashed_password",
		}

		err := repo.Create(user)

		assert.NoError(t, err, "ユーザー作成に失敗しました")
		assert.NotZero(t, user.ID, "IDが設定されていません")
		assert.False(t, user.CreatedAt.IsZero(), "CreatedAtが設定されていません")
		assert.False(t, user.UpdatedAt.IsZero(), "UpdatedAtが設定されていません")
	})

	t.Run("重複するメールアドレスでエラー", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		user1 := &entity.User{
			Email:    "duplicate@example.com",
			Password: "password1",
		}
		err := repo.Create(user1)
		require.NoError(t, err, "1つ目のユーザー作成に失敗しました")

		// 同じメールアドレスで2つ目のユーザーを作成
		user2 := &entity.User{
			Email:    "duplicate@example.com",
			Password: "password2",
		}
		err = repo.Create(user2)

		assert.Error(t, err, "重複エラーが発生するべきです")
	})

	t.Run("nilユーザーでエラー", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		err := repo.Create(nil)

		assert.Error(t, err, "nilユーザーでエラーが発生するべきです")
	})
}

func TestUserMySQL_FindByEmail(t *testing.T) {
	t.Run("メールアドレスでユーザーを検索成功", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		// テストデータを作成
		expected := &entity.User{
			Email:    "find@example.com",
			Password: "hashed_password",
		}
		err := repo.Create(expected)
		require.NoError(t, err, "テストデータの作成に失敗しました")

		// 検索実行
		found, err := repo.FindByEmail("find@example.com")

		assert.NoError(t, err, "ユーザーの検索に失敗しました")
		assert.NotNil(t, found, "ユーザーがnilです")
		assert.Equal(t, expected.ID, found.ID, "IDが一致しません")
		assert.Equal(t, expected.Email, found.Email, "メールアドレスが一致しません")
		assert.Equal(t, expected.Password, found.Password, "パスワードが一致しません")
	})

	t.Run("存在しないメールアドレスでエラー", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		found, err := repo.FindByEmail("notfound@example.com")

		assert.Error(t, err, "エラーが発生するべきです")
		assert.Nil(t, found, "ユーザーはnilであるべきです")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "ErrRecordNotFoundが返されるべきです")
	})

	t.Run("空のメールアドレスでエラー", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		found, err := repo.FindByEmail("")

		assert.Error(t, err, "エラーが発生するべきです")
		assert.Nil(t, found, "ユーザーはnilであるべきです")
	})

	t.Run("複数ユーザー存在時に正しいユーザーを検索", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		// 複数のユーザーを作成
		users := []*entity.User{
			{Email: "user1@example.com", Password: "pass1"},
			{Email: "user2@example.com", Password: "pass2"},
			{Email: "user3@example.com", Password: "pass3"},
		}
		for _, u := range users {
			err := repo.Create(u)
			require.NoError(t, err, "テストデータの作成に失敗しました")
		}

		// user2を検索
		found, err := repo.FindByEmail("user2@example.com")

		assert.NoError(t, err, "ユーザーの検索に失敗しました")
		assert.NotNil(t, found, "ユーザーがnilです")
		assert.Equal(t, users[1].ID, found.ID, "IDが一致しません")
		assert.Equal(t, "user2@example.com", found.Email, "メールアドレスが一致しません")
		assert.Equal(t, "pass2", found.Password, "パスワードが一致しません")
	})
}

func TestUserMySQL_FindByID(t *testing.T) {
	t.Run("IDでユーザーを検索成功", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		// テストデータを作成
		expected := &entity.User{
			Email:    "findbyid@example.com",
			Password: "hashed_password",
		}
		err := repo.Create(expected)
		require.NoError(t, err, "テストデータの作成に失敗しました")

		// 検索実行
		found, err := repo.FindByID(expected.ID)

		assert.NoError(t, err, "ユーザーの検索に失敗しました")
		assert.NotNil(t, found, "ユーザーがnilです")
		assert.Equal(t, expected.ID, found.ID, "IDが一致しません")
		assert.Equal(t, expected.Email, found.Email, "メールアドレスが一致しません")
		assert.Equal(t, expected.Password, found.Password, "パスワードが一致しません")
	})

	t.Run("存在しないIDでエラー", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		found, err := repo.FindByID(999)

		assert.Error(t, err, "エラーが発生するべきです")
		assert.Nil(t, found, "ユーザーはnilであるべきです")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "ErrRecordNotFoundが返されるべきです")
	})

	t.Run("ID 0でエラー", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		found, err := repo.FindByID(0)

		assert.Error(t, err, "エラーが発生するべきです")
		assert.Nil(t, found, "ユーザーはnilであるべきです")
	})

	t.Run("複数ユーザー存在時に正しいユーザーを検索", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		// 複数のユーザーを作成
		users := []*entity.User{
			{Email: "user1@example.com", Password: "pass1"},
			{Email: "user2@example.com", Password: "pass2"},
			{Email: "user3@example.com", Password: "pass3"},
		}
		for _, u := range users {
			err := repo.Create(u)
			require.NoError(t, err, "テストデータの作成に失敗しました")
		}

		// user2を検索
		found, err := repo.FindByID(users[1].ID)

		assert.NoError(t, err, "ユーザーの検索に失敗しました")
		assert.NotNil(t, found, "ユーザーがnilです")
		assert.Equal(t, users[1].ID, found.ID, "IDが一致しません")
		assert.Equal(t, "user2@example.com", found.Email, "メールアドレスが一致しません")
		assert.Equal(t, "pass2", found.Password, "パスワードが一致しません")
	})
}

func TestUserMySQL_Timestamps(t *testing.T) {
	t.Run("CreatedAtとUpdatedAtが自動設定される", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		beforeCreate := time.Now()
		user := &entity.User{
			Email:    "timestamp@example.com",
			Password: "password",
		}

		err := repo.Create(user)
		require.NoError(t, err, "ユーザー作成に失敗しました")

		afterCreate := time.Now()

		assert.False(t, user.CreatedAt.IsZero(), "CreatedAtが設定されていません")
		assert.False(t, user.UpdatedAt.IsZero(), "UpdatedAtが設定されていません")
		assert.True(t, user.CreatedAt.After(beforeCreate) || user.CreatedAt.Equal(beforeCreate),
			"CreatedAtが作成前の時刻より前です")
		assert.True(t, user.CreatedAt.Before(afterCreate) || user.CreatedAt.Equal(afterCreate),
			"CreatedAtが作成後の時刻より後です")

		// 検索後もタイムスタンプが保持されている
		found, err := repo.FindByID(user.ID)
		require.NoError(t, err, "ユーザー検索に失敗しました")

		assert.Equal(t, user.CreatedAt.Unix(), found.CreatedAt.Unix(), "CreatedAtが一致しません")
		assert.Equal(t, user.UpdatedAt.Unix(), found.UpdatedAt.Unix(), "UpdatedAtが一致しません")
	})
}
