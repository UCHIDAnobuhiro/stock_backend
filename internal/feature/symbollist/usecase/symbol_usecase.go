package usecase

import (
	"context"
	"stock_backend/internal/feature/symbollist/domain/entity"
)

// SymbolRepository はシンボル（銘柄）データの永続化を抽象化します。
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
	// 将来ここでバリデ/並び/絞り込みなどビジネスロジックを追加
	return u.repo.ListActive(ctx)
}
