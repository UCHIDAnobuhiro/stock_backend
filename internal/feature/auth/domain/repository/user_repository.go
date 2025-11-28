// Package repository defines repository interfaces for the auth feature.
package repository

import (
	"stock_backend/internal/feature/auth/domain/entity"
)

// UserRepository abstracts the persistence layer for user entities.
// It is used by the usecase layer and is independent of specific database implementations.
type UserRepository interface {
	// Create persists a new user to the storage.
	// It returns an error if a user with the same email already exists.
	Create(user *entity.User) error

	// FindByEmail retrieves a user matching the specified email address.
	// It returns an error if the user does not exist.
	FindByEmail(email string) (*entity.User, error)

	// FindByID retrieves a user matching the specified ID.
	// It returns an error if the user does not exist.
	FindByID(id uint) (*entity.User, error)
}
