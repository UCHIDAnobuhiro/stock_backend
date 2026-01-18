// internal/domain/entity/symbol.go
package entity

import "time"

type Symbol struct {
	ID        uint      `gorm:"primaryKey"`
	Code      string    `gorm:"size:20;not null;uniqueIndex"`
	Name      string    `gorm:"size:255;not null"`
	Market    string    `gorm:"size:100;not null"`
	IsActive  bool      `gorm:"not null;default:true"`
	SortKey   int       `gorm:"not null;default:0"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}
