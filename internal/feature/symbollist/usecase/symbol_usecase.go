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

type SymbolUsecase struct {
	repo SymbolRepository
}

func NewSymbolUsecase(r SymbolRepository) *SymbolUsecase {
	return &SymbolUsecase{repo: r}
}

func (u *SymbolUsecase) ListActiveSymbols(ctx context.Context) ([]entity.Symbol, error) {
	// Future enhancement: add validation, sorting, filtering, or other business logic here
	return u.repo.ListActive(ctx)
}
