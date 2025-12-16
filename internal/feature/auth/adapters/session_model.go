package adapters

import (
	"time"

	"stock_backend/internal/feature/auth/domain/entity"
)

// SessionModel is the GORM model for the sessions table.
type SessionModel struct {
	ID        string     `gorm:"primaryKey;size:64"`
	UserID    uint       `gorm:"index;not null"`
	UserAgent string     `gorm:"size:512"`
	IPAddress string     `gorm:"size:45"` // IPv6 max length
	CreatedAt time.Time  `gorm:"not null"`
	ExpiresAt time.Time  `gorm:"index;not null"`
	RevokedAt *time.Time `gorm:"index"`
}

// TableName returns the table name for GORM.
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

// SessionModelFromEntity converts a domain entity to a GORM model.
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
