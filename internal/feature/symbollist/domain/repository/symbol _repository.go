package repository

import (
	"context"
	"stock_backend/internal/feature/symbollist/domain/entity"
)

type SymbolRepository interface {
	ListActive(ctx context.Context) ([]entity.Symbol, error)

	ListActiveCodes(ctx context.Context) ([]string, error)
}
