// Package entity defines the domain entities for the auth feature.
package entity

import "time"

// User represents a registered user in the system.
// It contains authentication credentials and metadata for user management.
type User struct {
	// ID is the unique identifier for the user.
	ID uint `gorm:"primaryKey"`

	// Email is the user's email address used for authentication.
	// It must be unique across all users.
	Email string `gorm:"uniqueIndex;size:255;not null"`

	// Password is the hashed password for the user.
	// This should never store plaintext passwords.
	Password string `gorm:"size:255;not null"`

	// CreatedAt is the timestamp when the user was created.
	CreatedAt time.Time

	// UpdatedAt is the timestamp when the user was last updated.
	UpdatedAt time.Time
}
