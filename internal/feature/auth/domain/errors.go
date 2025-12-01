// Package domain defines domain-level errors for the auth feature.
package domain

import "errors"

// Domain errors for authentication operations.
// These errors represent business logic failures and should be handled appropriately by upper layers.
var (
	// ErrUserAlreadyExists indicates that a user with the given email already exists.
	// This is returned during signup when attempting to create a duplicate user.
	ErrUserAlreadyExists = errors.New("user with this email already exists")

	// ErrUserNotFound indicates that no user was found with the given criteria.
	// This is typically returned during login or user lookup operations.
	ErrUserNotFound = errors.New("user not found")

	// ErrInvalidCredentials indicates that the provided credentials are incorrect.
	// This is returned during login when email or password is invalid.
	ErrInvalidCredentials = errors.New("invalid email or password")
)
