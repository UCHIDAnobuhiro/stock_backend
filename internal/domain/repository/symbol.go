package repository

import (
	"context"
	"todo_backend/internal/domain/entity"
)

type SymbolRepository interface {
	ListActive(ctx context.Context) ([]entity.Symbol, error)
}
