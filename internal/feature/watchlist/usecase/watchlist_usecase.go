package usecase

import (
	"context"
	"fmt"

	"stock_backend/internal/feature/watchlist/domain/entity"
)

// DefaultSymbolCodes はサインアップ時に自動追加されるデフォルト銘柄のコードです。
var DefaultSymbolCodes = []string{"AAPL", "MSFT", "GOOGL"}

// UserSymbolRepository はユーザーのウォッチリスト銘柄の永続化レイヤーを抽象化します。
// Goの慣例に従い、インターフェースはプロバイダー（adapters）ではなくコンシューマー（usecase）が定義します。
type UserSymbolRepository interface {
	// ListByUser はユーザーのウォッチリスト銘柄をsort_key順に返します。
	ListByUser(ctx context.Context, userID uint) ([]entity.UserSymbol, error)
	// AddWithAtomicSortKey はsort_keyをトランザクション内でアトミックに採番しながら銘柄を追加します。
	// SELECT MAX(sort_key) FOR UPDATE + INSERT を1トランザクションで実行するため競合が発生しません。
	AddWithAtomicSortKey(ctx context.Context, userID uint, symbolCode string) error
	// Remove はユーザーのウォッチリストから銘柄を削除します。
	Remove(ctx context.Context, userID uint, symbolCode string) error
	// UpdateSortKeys はユーザーの銘柄の並び順を一括更新します。
	UpdateSortKeys(ctx context.Context, userID uint, codeOrder []string) error
	// AddBatch はユーザーのウォッチリストに複数の銘柄を一括追加します（デフォルト銘柄用）。
	AddBatch(ctx context.Context, userSymbols []entity.UserSymbol) error
}

// WatchlistUsecase はウォッチリスト操作のビジネスロジックを提供します。
type WatchlistUsecase struct {
	repo UserSymbolRepository
}

// NewWatchlistUsecase はWatchlistUsecaseの新しいインスタンスを生成します。
func NewWatchlistUsecase(r UserSymbolRepository) *WatchlistUsecase {
	return &WatchlistUsecase{repo: r}
}

// ListUserSymbols はユーザーのウォッチリスト銘柄をsort_key順に返します。
func (u *WatchlistUsecase) ListUserSymbols(ctx context.Context, userID uint) ([]entity.UserSymbol, error) {
	return u.repo.ListByUser(ctx, userID)
}

// AddSymbol はユーザーのウォッチリストに銘柄を追加します。
// sort_keyはリポジトリ内でアトミックに採番されます。
func (u *WatchlistUsecase) AddSymbol(ctx context.Context, userID uint, symbolCode string) error {
	if symbolCode == "" {
		return fmt.Errorf("symbol code must not be empty")
	}
	return u.repo.AddWithAtomicSortKey(ctx, userID, symbolCode)
}

// RemoveSymbol はユーザーのウォッチリストから銘柄を削除します。
func (u *WatchlistUsecase) RemoveSymbol(ctx context.Context, userID uint, symbolCode string) error {
	return u.repo.Remove(ctx, userID, symbolCode)
}

// ReorderSymbols はユーザーのウォッチリスト銘柄の並び順を更新します。
// codeOrderの順番に従い、sort_key = index * 10 で設定します。
// codeOrderは現在のウォッチリストと同じ銘柄集合・同じ件数・重複なしであることを検証します。
func (u *WatchlistUsecase) ReorderSymbols(ctx context.Context, userID uint, codeOrder []string) error {
	if len(codeOrder) == 0 {
		return ErrInvalidReorder
	}

	current, err := u.repo.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to load current watchlist: %w", err)
	}
	if len(current) != len(codeOrder) {
		return ErrInvalidReorder
	}

	allowed := make(map[string]struct{}, len(current))
	for _, s := range current {
		allowed[s.SymbolCode] = struct{}{}
	}
	seen := make(map[string]struct{}, len(codeOrder))
	for _, code := range codeOrder {
		if _, ok := allowed[code]; !ok {
			return ErrInvalidReorder
		}
		if _, dup := seen[code]; dup {
			return ErrInvalidReorder
		}
		seen[code] = struct{}{}
	}

	return u.repo.UpdateSortKeys(ctx, userID, codeOrder)
}

// InitializeDefaults はデフォルト銘柄をユーザーのウォッチリストに一括追加します。
// サインアップ時に呼び出されます。冪等性を保証します。
func (u *WatchlistUsecase) InitializeDefaults(ctx context.Context, userID uint) error {
	symbols := make([]entity.UserSymbol, len(DefaultSymbolCodes))
	for i, code := range DefaultSymbolCodes {
		symbols[i] = entity.UserSymbol{
			UserID:     userID,
			SymbolCode: code,
			SortKey:    (i + 1) * 10,
		}
	}
	return u.repo.AddBatch(ctx, symbols)
}
