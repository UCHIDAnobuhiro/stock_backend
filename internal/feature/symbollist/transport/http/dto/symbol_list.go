// Package dto defines data transfer objects for the symbollist HTTP API.
package dto

// SymbolItem represents a symbol in the API response.
// It contains only the public-facing fields needed by clients.
type SymbolItem struct {
	Code string `json:"code"`
	Name string `json:"name"`
}
