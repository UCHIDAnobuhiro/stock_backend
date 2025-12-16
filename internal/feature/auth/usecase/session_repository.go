package usecase

import (
	"context"

	"stock_backend/internal/feature/auth/domain/entity"
)

// SessionRepository abstracts the persistence layer for session entities.
// Following Go convention: interfaces are defined by the consumer (usecase), not the provider (adapters).
type SessionRepository interface {
	// Create persists a new session to the storage.
	Create(ctx context.Context, session *entity.Session) error

	// FindByID retrieves a session by its ID (refresh token value).
	FindByID(ctx context.Context, id string) (*entity.Session, error)

	// FindByUserID retrieves all sessions for a given user.
	FindByUserID(ctx context.Context, userID uint) ([]*entity.Session, error)

	// Revoke marks a session as revoked by setting RevokedAt.
	Revoke(ctx context.Context, id string) error

	// RevokeAllByUserID revokes all sessions for a given user.
	RevokeAllByUserID(ctx context.Context, userID uint) error

	// DeleteExpired removes all expired sessions from storage.
	// Returns the number of deleted sessions.
	DeleteExpired(ctx context.Context) (int64, error)

	// CountByUserID returns the number of active sessions for a user.
	CountByUserID(ctx context.Context, userID uint) (int64, error)

	// DeleteOldestByUserID deletes the oldest session for a user.
	DeleteOldestByUserID(ctx context.Context, userID uint) error
}
