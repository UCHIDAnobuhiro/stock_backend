// Package session provides Redis implementation for session storage.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"stock_backend/internal/feature/auth/domain/entity"
	"stock_backend/internal/feature/auth/usecase"
)

// sessionRedis is a Redis implementation of the SessionRepository interface.
type sessionRedis struct {
	client *redis.Client
	prefix string
}

// Compile-time check to ensure sessionRedis implements SessionRepository.
var _ usecase.SessionRepository = (*sessionRedis)(nil)

// NewSessionRedis creates a new instance of sessionRedis.
// namespace is used as a prefix for Redis keys (e.g., "session").
func NewSessionRedis(client *redis.Client, prefix string) *sessionRedis {
	return &sessionRedis{
		client: client,
		prefix: prefix,
	}
}

// Key formats:
// - Session data: {prefix}:{id} -> JSON
// - User sessions set: {prefix}:user:{userID} -> Set of session IDs

// sessionKey returns the Redis key for a session.
func (r *sessionRedis) sessionKey(id string) string {
	return fmt.Sprintf("%s:%s", r.prefix, id)
}

// userSessionsKey returns the Redis key for a user's session set.
func (r *sessionRedis) userSessionsKey(userID uint) string {
	return fmt.Sprintf("%s:user:%d", r.prefix, userID)
}

// Create persists a new session to Redis.
func (r *sessionRedis) Create(ctx context.Context, session *entity.Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("session already expired")
	}

	if err := r.client.Set(ctx, r.sessionKey(session.ID), data, ttl).Err(); err != nil {
		return err
	}

	return r.client.SAdd(ctx, r.userSessionsKey(session.UserID), session.ID).Err()
}

// FindByID retrieves a session by its refresh token ID.
// Returns usecase.ErrSessionNotFound if the session does not exist.
func (r *sessionRedis) FindByID(ctx context.Context, id string) (*entity.Session, error) {
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

// FindByUserID retrieves all active sessions for a given user.
// Automatically cleans up expired session IDs from the user's session set.
func (r *sessionRedis) FindByUserID(ctx context.Context, userID uint) ([]*entity.Session, error) {
	ids, err := r.client.SMembers(ctx, r.userSessionsKey(userID)).Result()
	if err != nil {
		return nil, err
	}

	var sessions []*entity.Session
	for _, id := range ids {
		session, err := r.FindByID(ctx, id)
		if err != nil {
			if err == usecase.ErrSessionNotFound {
				// Clean up stale session ID from user's set
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

// Revoke marks a session as revoked by its ID.
// The revoked session is kept for 24 hours for audit purposes.
func (r *sessionRedis) Revoke(ctx context.Context, id string) error {
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
	return r.client.Set(ctx, r.sessionKey(id), data, 24*time.Hour).Err()
}

// RevokeAllByUserID revokes all sessions for a given user.
func (r *sessionRedis) RevokeAllByUserID(ctx context.Context, userID uint) error {
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

// DeleteExpired removes all expired sessions from storage.
// Note: Redis TTL handles session expiration automatically.
// This method is a no-op for Redis implementation.
func (r *sessionRedis) DeleteExpired(ctx context.Context) (int64, error) {
	return 0, nil
}

// CountByUserID returns the number of active sessions for a user.
func (r *sessionRedis) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	sessions, err := r.FindByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	return int64(len(sessions)), nil
}

// DeleteOldestByUserID deletes the oldest active session for a user.
// Used to enforce MaxSessionsPerUser limit.
func (r *sessionRedis) DeleteOldestByUserID(ctx context.Context, userID uint) error {
	sessions, err := r.FindByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		return nil
	}

	oldest := sessions[0]
	for _, s := range sessions[1:] {
		if s.CreatedAt.Before(oldest.CreatedAt) {
			oldest = s
		}
	}

	if err := r.client.Del(ctx, r.sessionKey(oldest.ID)).Err(); err != nil {
		return err
	}
	return r.client.SRem(ctx, r.userSessionsKey(userID), oldest.ID).Err()
}

