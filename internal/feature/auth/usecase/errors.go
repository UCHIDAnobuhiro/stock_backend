// Package usecase implements the business logic for the auth feature.
package usecase

import "errors"

var (
	// ErrUserNotFound is returned when a user cannot be found by email or ID.
	ErrUserNotFound = errors.New("user not found")

	// ErrEmailAlreadyExists is returned when attempting to create a user with an email that already exists.
	ErrEmailAlreadyExists = errors.New("email already exists")
)
