// Package entity defines the domain entities for the auth feature.
package entity

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

const (
	// RefreshTokenLength is the byte length of the refresh token (32 bytes = 256 bits).
	RefreshTokenLength = 32

	// RefreshTokenExpiry is the default expiration duration for refresh tokens.
	RefreshTokenExpiry = 7 * 24 * time.Hour // 7 days

	// MaxSessionsPerUser is the maximum number of concurrent sessions allowed per user.
	MaxSessionsPerUser = 5
)

// Session represents a user's refresh token session.
// It contains metadata for session management and security.
type Session struct {
	// ID is the refresh token value (64-character hex string).
	ID string

	// UserID is the ID of the user who owns this session.
	UserID uint

	// UserAgent is the browser/client identifier.
	UserAgent string

	// IPAddress is the client's IP address.
	IPAddress string

	// CreatedAt is when the session was created.
	CreatedAt time.Time

	// ExpiresAt is when the session expires.
	ExpiresAt time.Time

	// RevokedAt is when the session was revoked (nil if active).
	RevokedAt *time.Time
}

// NewSession creates a new session with a cryptographically secure refresh token.
func NewSession(userID uint, userAgent, ipAddress string) (*Session, error) {
	tokenBytes := make([]byte, RefreshTokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}

	now := time.Now()
	return &Session{
		ID:        hex.EncodeToString(tokenBytes),
		UserID:    userID,
		UserAgent: userAgent,
		IPAddress: ipAddress,
		CreatedAt: now,
		ExpiresAt: now.Add(RefreshTokenExpiry),
		RevokedAt: nil,
	}, nil
}

// IsValid returns true if the session is both not expired and not revoked.
func (s *Session) IsValid() bool {
	return !s.IsExpired() && !s.IsRevoked()
}

// IsExpired returns true if the session has passed its expiration time.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsRevoked returns true if the session has been explicitly revoked.
func (s *Session) IsRevoked() bool {
	return s.RevokedAt != nil
}

// Revoke marks the session as revoked at the current time.
func (s *Session) Revoke() {
	now := time.Now()
	s.RevokedAt = &now
}
