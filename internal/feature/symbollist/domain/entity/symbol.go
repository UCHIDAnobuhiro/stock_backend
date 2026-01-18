// Package entity defines the domain models for the symbollist feature.
package entity

import "time"

// Symbol represents a stock ticker symbol in the system.
// It contains information about a tradable security including its code,
// name, market, and display ordering.
type Symbol struct {
	ID        uint      `gorm:"primaryKey"`
	Code      string    `gorm:"size:20;not null;uniqueIndex"`
	Name      string    `gorm:"size:255;not null"`
	Market    string    `gorm:"size:100;not null"`
	IsActive  bool      `gorm:"not null;default:true"`
	SortKey   int       `gorm:"not null;default:0"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}
