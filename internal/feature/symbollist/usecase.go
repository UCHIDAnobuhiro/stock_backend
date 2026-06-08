package symbollist

import (
	"context"
)

// Repository は銘柄（株式コード）データの永続化レイヤーを抽象化します。
// Goの慣例に従い、インターフェースは利用者（usecase）側で定義します。
type Repository interface {
	// ListActive はすべてのアクティブな銘柄を返します。
	ListActive(ctx context.Context) ([]Symbol, error)
}

// usecase は銘柄操作のビジネスロジックを提供します。
type usecase struct {
	repo Repository
}

// NewUsecase は指定されたリポジトリでusecaseの新しいインスタンスを生成します。
func NewUsecase(r Repository) *usecase {
	return &usecase{repo: r}
}

// ListActiveSymbols はリポジトリからすべてのアクティブな銘柄を取得して返します。
func (u *usecase) ListActiveSymbols(ctx context.Context) ([]Symbol, error) {
	return u.repo.ListActive(ctx)
}
