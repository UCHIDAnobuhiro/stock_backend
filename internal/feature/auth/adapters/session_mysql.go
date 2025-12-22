// Package adapters provides repository implementations for the auth feature.
package adapters

import (
	"context"
	"errors"
	"time"

	"stock_backend/internal/feature/auth/domain/entity"
	"stock_backend/internal/feature/auth/usecase"

	"gorm.io/gorm"
)

// sessionMySQL is a MySQL implementation of the SessionRepository interface.
type sessionMySQL struct {
	db *gorm.DB
}

// Compile-time check to ensure sessionMySQL implements SessionRepository.
var _ usecase.SessionRepository = (*sessionMySQL)(nil)

// NewSessionMySQL creates a new instance of sessionMySQL.
func NewSessionMySQL(db *gorm.DB) *sessionMySQL {
	return &sessionMySQL{db: db}
}

// Create persists a new session to the database.
func (r *sessionMySQL) Create(ctx context.Context, session *entity.Session) error {
	model := SessionModelFromEntity(session)
	return r.db.WithContext(ctx).Create(model).Error
}

// FindByID retrieves a session by its refresh token ID.
func (r *sessionMySQL) FindByID(ctx context.Context, id string) (*entity.Session, error) {
	var model SessionModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, usecase.ErrSessionNotFound
		}
		return nil, err
	}
	return model.ToEntity(), nil
}

// FindByUserID retrieves all active sessions for a given user.
func (r *sessionMySQL) FindByUserID(ctx context.Context, userID uint) ([]*entity.Session, error) {
	var models []SessionModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Order("created_at ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}

	sessions := make([]*entity.Session, len(models))
	for i, m := range models {
		sessions[i] = m.ToEntity()
	}
	return sessions, nil
}

// Revoke marks a session as revoked by its ID.
func (r *sessionMySQL) Revoke(ctx context.Context, id string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&SessionModel{}).
		Where("id = ?", id).
		Update("revoked_at", now)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return usecase.ErrSessionNotFound
	}
	return nil
}

// RevokeAllByUserID revokes all sessions for a given user.
func (r *sessionMySQL) RevokeAllByUserID(ctx context.Context, userID uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&SessionModel{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

// DeleteExpired removes all expired sessions from storage.
func (r *sessionMySQL) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&SessionModel{})
	return result.RowsAffected, result.Error
}

// CountByUserID returns the number of active sessions for a user.
func (r *sessionMySQL) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&SessionModel{}).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Count(&count).Error
	return count, err
}

// DeleteOldestByUserID deletes the oldest active session for a user.
func (r *sessionMySQL) DeleteOldestByUserID(ctx context.Context, userID uint) error {
	// Find the oldest session
	var oldest SessionModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Order("created_at ASC").
		First(&oldest).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // No sessions to delete
		}
		return err
	}

	// Delete it
	return r.db.WithContext(ctx).Delete(&SessionModel{}, "id = ?", oldest.ID).Error
}
