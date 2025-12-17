// Package usecase implements the business logic for the auth feature.
package usecase

import (
	"context"

	"stock_backend/internal/feature/auth/domain/entity"
)

// SessionRepository abstracts the persistence layer for session entities.
// Following Go convention: interfaces are defined by the consumer (usecase), not the provider (adapters).
// This interface can be implemented by both MySQL and Redis backends.
type SessionRepository interface {
	// Create persists a new session to the storage.
	Create(ctx context.Context, session *entity.Session) error

	// FindByID retrieves a session by its refresh token ID.
	// Returns ErrSessionNotFound if the session does not exist.
	FindByID(ctx context.Context, id string) (*entity.Session, error)

	// FindByUserID retrieves all active sessions for a given user.
	// Returns an empty slice if no sessions exist.
	FindByUserID(ctx context.Context, userID uint) ([]*entity.Session, error)

	// Revoke marks a session as revoked by its ID.
	// Returns ErrSessionNotFound if the session does not exist.
	Revoke(ctx context.Context, id string) error

	// RevokeAllByUserID revokes all sessions for a given user.
	RevokeAllByUserID(ctx context.Context, userID uint) error

	// DeleteExpired removes all expired sessions from storage.
	// Returns the number of deleted sessions.
	DeleteExpired(ctx context.Context) (int64, error)

	// CountByUserID returns the number of active sessions for a user.
	CountByUserID(ctx context.Context, userID uint) (int64, error)

	// DeleteOldestByUserID deletes the oldest active session for a user.
	// Used to enforce MaxSessionsPerUser limit.
	DeleteOldestByUserID(ctx context.Context, userID uint) error
}
