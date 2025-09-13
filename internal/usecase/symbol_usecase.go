package usecase

import (
	"context"
	"todo_backend/internal/domain/entity"
	"todo_backend/internal/domain/repository"
)

type SymbolUsecase struct {
	repo repository.SymbolRepository
}

func NewSymbolUsecase(r repository.SymbolRepository) *SymbolUsecase {
	return &SymbolUsecase{repo: r}
}

func (u *SymbolUsecase) ListActiveSymbols(ctx context.Context) ([]entity.Symbol, error) {
	// 将来ここでバリデ/並び/絞り込みなどビジネスロジックを追加
	return u.repo.ListActive(ctx)
}
