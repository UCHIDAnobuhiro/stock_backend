// Package adapters provides repository implementations for the auth feature.
package adapters

import (
	"time"

	"stock_backend/internal/feature/auth/domain/entity"
)

// SessionModel is the GORM model for session persistence.
type SessionModel struct {
	ID        string `gorm:"primaryKey;size:64"`
	UserID    uint   `gorm:"index;not null"`
	UserAgent string `gorm:"size:512"`
	IPAddress string `gorm:"size:45"`
	CreatedAt time.Time
	ExpiresAt time.Time `gorm:"index"`
	RevokedAt *time.Time
}

// TableName specifies the table name for GORM.
func (SessionModel) TableName() string {
	return "sessions"
}

// ToEntity converts the GORM model to a domain entity.
func (m *SessionModel) ToEntity() *entity.Session {
	return &entity.Session{
		ID:        m.ID,
		UserID:    m.UserID,
		UserAgent: m.UserAgent,
		IPAddress: m.IPAddress,
		CreatedAt: m.CreatedAt,
		ExpiresAt: m.ExpiresAt,
		RevokedAt: m.RevokedAt,
	}
}

// SessionModelFromEntity creates a GORM model from a domain entity.
func SessionModelFromEntity(s *entity.Session) *SessionModel {
	return &SessionModel{
		ID:        s.ID,
		UserID:    s.UserID,
		UserAgent: s.UserAgent,
		IPAddress: s.IPAddress,
		CreatedAt: s.CreatedAt,
		ExpiresAt: s.ExpiresAt,
		RevokedAt: s.RevokedAt,
	}
}
