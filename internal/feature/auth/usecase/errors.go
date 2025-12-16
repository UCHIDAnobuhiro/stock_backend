// Package usecase implements the business logic for the auth feature.
package usecase

import "errors"

var (
	// ErrUserNotFound is returned when a user cannot be found by email or ID.
	ErrUserNotFound = errors.New("user not found")

	// ErrEmailAlreadyExists is returned when attempting to create a user with an email that already exists.
	ErrEmailAlreadyExists = errors.New("email already exists")

	// ErrSessionNotFound is returned when a session cannot be found by ID.
	ErrSessionNotFound = errors.New("session not found")

	// ErrSessionRevoked is returned when attempting to use a revoked session.
	ErrSessionRevoked = errors.New("session has been revoked")

	// ErrSessionExpired is returned when attempting to use an expired session.
	ErrSessionExpired = errors.New("session has expired")

	// ErrInvalidRefreshToken is returned when a refresh token is invalid or malformed.
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)
