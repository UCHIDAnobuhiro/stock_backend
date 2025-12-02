// Package adapters provides repository implementations for the auth feature.
package adapters

import (
	"context"
	"errors"
	"strings"

	"stock_backend/internal/feature/auth/domain/entity"
	"stock_backend/internal/feature/auth/usecase"

	"gorm.io/gorm"
)

// userMySQL is a MySQL implementation of the UserRepository interface.
// It uses GORM to perform database operations.
type userMySQL struct {
	db *gorm.DB
}

// Compile-time check to ensure userMySQL implements UserRepository.
var _ usecase.UserRepository = (*userMySQL)(nil)

// NewUserMySQL creates a new instance of userMySQL with the given gorm.DB connection.
// This is a constructor for dependency injection.
func NewUserMySQL(db *gorm.DB) *userMySQL {
	return &userMySQL{db: db}
}

// Create adds a user to the database.
// Returns usecase.ErrEmailAlreadyExists if a user with the same email already exists.
func (r *userMySQL) Create(ctx context.Context, u *entity.User) error {
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		// MySQL error 1062: Duplicate entry for unique key
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "duplicate key") {
			return usecase.ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}

// FindByEmail retrieves a user by email address.
// Returns usecase.ErrUserNotFound if the user does not exist.
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

// FindByID retrieves a user by ID.
// Returns usecase.ErrUserNotFound if the user does not exist.
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
