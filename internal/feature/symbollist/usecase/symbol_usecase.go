// Package usecase implements the business logic for symbol-related operations.
package usecase

import (
	"context"
	"stock_backend/internal/feature/symbollist/domain/entity"
)

// SymbolRepository abstracts the persistence layer for symbol (stock ticker) data.
// Following Go convention: interfaces are defined by the consumer (usecase), not the provider (adapters).
type SymbolRepository interface {
	ListActive(ctx context.Context) ([]entity.Symbol, error)
	ListActiveCodes(ctx context.Context) ([]string, error)
}

// SymbolUsecase provides business logic for symbol operations.
type SymbolUsecase struct {
	repo SymbolRepository
}

// NewSymbolUsecase creates a new SymbolUsecase with the given repository.
func NewSymbolUsecase(r SymbolRepository) *SymbolUsecase {
	return &SymbolUsecase{repo: r}
}

// ListActiveSymbols returns all active symbols from the repository.
func (u *SymbolUsecase) ListActiveSymbols(ctx context.Context) ([]entity.Symbol, error) {
	// Future enhancement: add validation, sorting, filtering, or other business logic here
	return u.repo.ListActive(ctx)
}
