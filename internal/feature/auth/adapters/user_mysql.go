// Package adapters provides repository implementations for the auth feature.
package adapters

import (
	"stock_backend/internal/feature/auth/domain/entity"
	"stock_backend/internal/feature/auth/domain/repository"

	"gorm.io/gorm"
)

// userMySQL is a MySQL implementation of the UserRepository interface.
// It uses GORM to perform database operations.
type userMySQL struct {
	db *gorm.DB
}

// Compile-time check to ensure userMySQL implements UserRepository.
var _ repository.UserRepository = (*userMySQL)(nil)

// NewUserMySQL creates a new instance of userMySQL with the given gorm.DB connection.
// This is a constructor for dependency injection.
func NewUserMySQL(db *gorm.DB) *userMySQL {
	return &userMySQL{db: db}
}

// Create adds a user to the database.
func (r *userMySQL) Create(u *entity.User) error {
	return r.db.Create(u).Error
}

// FindByEmail retrieves a user by email address.
// It returns an error if the user does not exist.
func (r *userMySQL) FindByEmail(email string) (*entity.User, error) {
	var u entity.User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByID retrieves a user by ID.
// It returns an error if the user does not exist.
func (r *userMySQL) FindByID(id uint) (*entity.User, error) {
	var u entity.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}
