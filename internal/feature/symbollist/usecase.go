package symbollist

import (
	"context"
)

// SymbolRepository は銘柄（株式コード）データの永続化レイヤーを抽象化します。
// Goの慣例に従い、インターフェースは利用者（usecase）側で定義します。
type SymbolRepository interface {
	// ListActive はすべてのアクティブな銘柄を返します。
	ListActive(ctx context.Context) ([]Symbol, error)
}

// SymbolUsecase は銘柄操作のビジネスロジックを提供します。
type SymbolUsecase struct {
	repo SymbolRepository
}

// NewSymbolUsecase は指定されたリポジトリでSymbolUsecaseの新しいインスタンスを生成します。
func NewSymbolUsecase(r SymbolRepository) *SymbolUsecase {
	return &SymbolUsecase{repo: r}
}

// ListActiveSymbols はリポジトリからすべてのアクティブな銘柄を取得して返します。
func (u *SymbolUsecase) ListActiveSymbols(ctx context.Context) ([]Symbol, error) {
	return u.repo.ListActive(ctx)
}
