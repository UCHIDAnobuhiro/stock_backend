package adapters

import (
	"context"
	"stock_backend/internal/feature/auth/domain/entity"
	"stock_backend/internal/feature/auth/usecase"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB prepares an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to initialize test database")

	// Create User table
	err = db.AutoMigrate(&entity.User{})
	require.NoError(t, err, "failed to migrate table")

	return db
}

func TestNewUserMySQL(t *testing.T) {
	db := setupTestDB(t)

	repo := NewUserMySQL(db)

	assert.NotNil(t, repo, "repository is nil")
	assert.NotNil(t, repo.db, "database connection is nil")
}

func TestUserMySQL_Create(t *testing.T) {
	t.Run("successful user creation", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		user := &entity.User{
			Email:    "test@example.com",
			Password: "hashed_password",
		}

		err := repo.Create(context.Background(), user)

		assert.NoError(t, err, "failed to create user")
		assert.NotZero(t, user.ID, "ID is not set")
		assert.False(t, user.CreatedAt.IsZero(), "CreatedAt is not set")
		assert.False(t, user.UpdatedAt.IsZero(), "UpdatedAt is not set")
	})

	t.Run("duplicate email error", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		user1 := &entity.User{
			Email:    "duplicate@example.com",
			Password: "password1",
		}
		err := repo.Create(context.Background(), user1)
		require.NoError(t, err, "failed to create first user")

		// Create second user with the same email
		user2 := &entity.User{
			Email:    "duplicate@example.com",
			Password: "password2",
		}
		err = repo.Create(context.Background(), user2)

		assert.Error(t, err, "should return duplicate error")
	})

	t.Run("nil user error", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		err := repo.Create(context.Background(), nil)

		assert.Error(t, err, "should return error for nil user")
	})
}

func TestUserMySQL_FindByEmail(t *testing.T) {
	t.Run("find user by email successfully", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		// Create test data
		expected := &entity.User{
			Email:    "find@example.com",
			Password: "hashed_password",
		}
		err := repo.Create(context.Background(), expected)
		require.NoError(t, err, "failed to create test data")

		// Execute search
		found, err := repo.FindByEmail(context.Background(), "find@example.com")

		assert.NoError(t, err, "failed to find user")
		assert.NotNil(t, found, "user is nil")
		assert.Equal(t, expected.ID, found.ID, "ID does not match")
		assert.Equal(t, expected.Email, found.Email, "email does not match")
		assert.Equal(t, expected.Password, found.Password, "password does not match")
	})

	t.Run("email not found error", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		found, err := repo.FindByEmail(context.Background(), "notfound@example.com")

		assert.Error(t, err, "should return error")
		assert.Nil(t, found, "user should be nil")
		assert.ErrorIs(t, err, usecase.ErrUserNotFound, "should return ErrUserNotFound")
	})

	t.Run("empty email error", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		found, err := repo.FindByEmail(context.Background(), "")

		assert.Error(t, err, "should return error")
		assert.Nil(t, found, "user should be nil")
	})

	t.Run("find correct user when multiple users exist", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		// Create multiple users
		users := []*entity.User{
			{Email: "user1@example.com", Password: "pass1"},
			{Email: "user2@example.com", Password: "pass2"},
			{Email: "user3@example.com", Password: "pass3"},
		}
		for _, u := range users {
			err := repo.Create(context.Background(), u)
			require.NoError(t, err, "failed to create test data")
		}

		// Find user2
		found, err := repo.FindByEmail(context.Background(), "user2@example.com")

		assert.NoError(t, err, "failed to find user")
		assert.NotNil(t, found, "user is nil")
		assert.Equal(t, users[1].ID, found.ID, "ID does not match")
		assert.Equal(t, "user2@example.com", found.Email, "email does not match")
		assert.Equal(t, "pass2", found.Password, "password does not match")
	})
}

func TestUserMySQL_FindByID(t *testing.T) {
	t.Run("find user by ID successfully", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		// Create test data
		expected := &entity.User{
			Email:    "findbyid@example.com",
			Password: "hashed_password",
		}
		err := repo.Create(context.Background(), expected)
		require.NoError(t, err, "failed to create test data")

		// Execute search
		found, err := repo.FindByID(context.Background(), expected.ID)

		assert.NoError(t, err, "failed to find user")
		assert.NotNil(t, found, "user is nil")
		assert.Equal(t, expected.ID, found.ID, "ID does not match")
		assert.Equal(t, expected.Email, found.Email, "email does not match")
		assert.Equal(t, expected.Password, found.Password, "password does not match")
	})

	t.Run("ID not found error", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		found, err := repo.FindByID(context.Background(), 999)

		assert.Error(t, err, "should return error")
		assert.Nil(t, found, "user should be nil")
		assert.ErrorIs(t, err, usecase.ErrUserNotFound, "should return ErrUserNotFound")
	})

	t.Run("ID 0 error", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		found, err := repo.FindByID(context.Background(), 0)

		assert.Error(t, err, "should return error")
		assert.Nil(t, found, "user should be nil")
	})

	t.Run("find correct user when multiple users exist", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		// Create multiple users
		users := []*entity.User{
			{Email: "user1@example.com", Password: "pass1"},
			{Email: "user2@example.com", Password: "pass2"},
			{Email: "user3@example.com", Password: "pass3"},
		}
		for _, u := range users {
			err := repo.Create(context.Background(), u)
			require.NoError(t, err, "failed to create test data")
		}

		// Find user2
		found, err := repo.FindByID(context.Background(), users[1].ID)

		assert.NoError(t, err, "failed to find user")
		assert.NotNil(t, found, "user is nil")
		assert.Equal(t, users[1].ID, found.ID, "ID does not match")
		assert.Equal(t, "user2@example.com", found.Email, "email does not match")
		assert.Equal(t, "pass2", found.Password, "password does not match")
	})
}

func TestUserMySQL_Timestamps(t *testing.T) {
	t.Run("CreatedAt and UpdatedAt are automatically set", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserMySQL(db)

		beforeCreate := time.Now()
		user := &entity.User{
			Email:    "timestamp@example.com",
			Password: "password",
		}

		err := repo.Create(context.Background(), user)
		require.NoError(t, err, "failed to create user")

		afterCreate := time.Now()

		assert.False(t, user.CreatedAt.IsZero(), "CreatedAt is not set")
		assert.False(t, user.UpdatedAt.IsZero(), "UpdatedAt is not set")
		assert.True(t, user.CreatedAt.After(beforeCreate) || user.CreatedAt.Equal(beforeCreate),
			"CreatedAt is before creation time")
		assert.True(t, user.CreatedAt.Before(afterCreate) || user.CreatedAt.Equal(afterCreate),
			"CreatedAt is after creation time")

		// Timestamps are preserved after retrieval
		found, err := repo.FindByID(context.Background(), user.ID)
		require.NoError(t, err, "failed to find user")

		assert.Equal(t, user.CreatedAt.Unix(), found.CreatedAt.Unix(), "CreatedAt does not match")
		assert.Equal(t, user.UpdatedAt.Unix(), found.UpdatedAt.Unix(), "UpdatedAt does not match")
	})
}
