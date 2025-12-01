// Package adapters provides repository implementations for the auth feature.
package adapters

import (
	"errors"

	"stock_backend/internal/feature/auth/domain"
	"stock_backend/internal/feature/auth/domain/entity"
	"stock_backend/internal/feature/auth/usecase"

	"github.com/go-sql-driver/mysql"
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
// It returns domain.ErrUserAlreadyExists if a user with the same email already exists.
func (r *userMySQL) Create(u *entity.User) error {
	err := r.db.Create(u).Error
	if err != nil {
		// Check for MySQL duplicate entry error (error code 1062)
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return domain.ErrUserAlreadyExists
		}
		// Check for SQLite UNIQUE constraint error (for testing)
		// SQLite returns "UNIQUE constraint failed" in the error message
		errMsg := err.Error()
		if errMsg == "UNIQUE constraint failed: users.email" {
			return domain.ErrUserAlreadyExists
		}
		return err
	}
	return nil
}

// FindByEmail retrieves a user by email address.
// It returns domain.ErrUserNotFound if the user does not exist.
func (r *userMySQL) FindByEmail(email string) (*entity.User, error) {
	var u entity.User
	err := r.db.Where("email = ?", email).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// FindByID retrieves a user by ID.
// It returns domain.ErrUserNotFound if the user does not exist.
func (r *userMySQL) FindByID(id uint) (*entity.User, error) {
	var u entity.User
	err := r.db.First(&u, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}
