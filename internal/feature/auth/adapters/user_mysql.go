// Package adapters provides repository implementations for the auth feature.
package adapters

import (
	"context"

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
func (r *userMySQL) Create(ctx context.Context, u *entity.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

// FindByEmail retrieves a user by email address.
// It returns an error if the user does not exist.
func (r *userMySQL) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var u entity.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByID retrieves a user by ID.
// It returns an error if the user does not exist.
func (r *userMySQL) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	var u entity.User
	if err := r.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}
