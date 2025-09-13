// internal/domain/entity/symbol.go
package entity

import "time"

type Symbol struct {
	Code      string
	Name      string
	Market    string
	IsActive  bool
	SortKey   int
	UpdatedAt time.Time
}
