package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"stock_backend/internal/feature/auth/domain/entity"
	"stock_backend/internal/feature/auth/usecase"

	"github.com/redis/go-redis/v9"
)

// SessionRedis implements usecase.SessionRepository using Redis.
type SessionRedis struct {
	client *redis.Client
	prefix string
}

// NewSessionRedis creates a new SessionRedis instance.
func NewSessionRedis(client *redis.Client, prefix string) *SessionRedis {
	return &SessionRedis{
		client: client,
		prefix: prefix,
	}
}

// sessionKey returns the Redis key for a session.
func (r *SessionRedis) sessionKey(id string) string {
	return fmt.Sprintf("%s:%s", r.prefix, id)
}

// userSessionsKey returns the Redis key for a user's session set.
func (r *SessionRedis) userSessionsKey(userID uint) string {
	return fmt.Sprintf("%s:user:%d", r.prefix, userID)
}

// Create persists a new session to Redis.
func (r *SessionRedis) Create(ctx context.Context, session *entity.Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("session already expired")
	}

	// Store session data
	if err := r.client.Set(ctx, r.sessionKey(session.ID), data, ttl).Err(); err != nil {
		return err
	}

	// Add to user's session set
	if err := r.client.SAdd(ctx, r.userSessionsKey(session.UserID), session.ID).Err(); err != nil {
		return err
	}

	return nil
}

// FindByID retrieves a session by its ID.
func (r *SessionRedis) FindByID(ctx context.Context, id string) (*entity.Session, error) {
	data, err := r.client.Get(ctx, r.sessionKey(id)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, usecase.ErrSessionNotFound
		}
		return nil, err
	}

	var session entity.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// FindByUserID retrieves all active sessions for a user.
func (r *SessionRedis) FindByUserID(ctx context.Context, userID uint) ([]*entity.Session, error) {
	// Get all session IDs for user
	ids, err := r.client.SMembers(ctx, r.userSessionsKey(userID)).Result()
	if err != nil {
		return nil, err
	}

	var sessions []*entity.Session
	for _, id := range ids {
		session, err := r.FindByID(ctx, id)
		if err != nil {
			if err == usecase.ErrSessionNotFound {
				// Session expired, remove from set
				r.client.SRem(ctx, r.userSessionsKey(userID), id)
				continue
			}
			return nil, err
		}
		if session.IsValid() {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// Revoke marks a session as revoked.
func (r *SessionRedis) Revoke(ctx context.Context, id string) error {
	session, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now()
	session.RevokedAt = &now

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Keep short TTL for revoked sessions (for audit purposes)
	return r.client.Set(ctx, r.sessionKey(id), data, 24*time.Hour).Err()
}

// RevokeAllByUserID revokes all sessions for a user.
func (r *SessionRedis) RevokeAllByUserID(ctx context.Context, userID uint) error {
	ids, err := r.client.SMembers(ctx, r.userSessionsKey(userID)).Result()
	if err != nil {
		return err
	}

	for _, id := range ids {
		if err := r.Revoke(ctx, id); err != nil && err != usecase.ErrSessionNotFound {
			return err
		}
	}

	return nil
}

// DeleteExpired removes expired sessions (handled by Redis TTL).
func (r *SessionRedis) DeleteExpired(ctx context.Context) (int64, error) {
	// Redis handles expiration automatically via TTL
	return 0, nil
}

// CountByUserID returns the number of active sessions for a user.
func (r *SessionRedis) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	sessions, err := r.FindByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	return int64(len(sessions)), nil
}

// DeleteOldestByUserID deletes the oldest session for a user.
func (r *SessionRedis) DeleteOldestByUserID(ctx context.Context, userID uint) error {
	sessions, err := r.FindByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		return nil
	}

	// Find oldest session
	oldest := sessions[0]
	for _, s := range sessions[1:] {
		if s.CreatedAt.Before(oldest.CreatedAt) {
			oldest = s
		}
	}

	// Delete from Redis
	if err := r.client.Del(ctx, r.sessionKey(oldest.ID)).Err(); err != nil {
		return err
	}

	// Remove from user's session set
	return r.client.SRem(ctx, r.userSessionsKey(userID), oldest.ID).Err()
}
