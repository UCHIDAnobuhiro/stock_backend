package entity

import "time"

// Session represents a user's authentication session (refresh token).
// It stores session metadata for token management and security auditing.
type Session struct {
	ID        string     // Refresh token value (64-character hex string)
	UserID    uint       // Associated user ID
	UserAgent string     // Client's User-Agent header
	IPAddress string     // Client's IP address
	CreatedAt time.Time  // Session creation time
	ExpiresAt time.Time  // Session expiration time
	RevokedAt *time.Time // Revocation time (nil if active)
}

// IsExpired returns true if the session has passed its expiration time.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsRevoked returns true if the session has been revoked.
func (s *Session) IsRevoked() bool {
	return s.RevokedAt != nil
}

// IsValid returns true if the session is neither expired nor revoked.
func (s *Session) IsValid() bool {
	return !s.IsExpired() && !s.IsRevoked()
}
