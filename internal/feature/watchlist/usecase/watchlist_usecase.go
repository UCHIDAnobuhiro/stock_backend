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
	// Add はユーザーのウォッチリストに銘柄を追加します。
	Add(ctx context.Context, userSymbol *entity.UserSymbol) error
	// Remove はユーザーのウォッチリストから銘柄を削除します。
	Remove(ctx context.Context, userID uint, symbolCode string) error
	// UpdateSortKeys はユーザーの銘柄の並び順を一括更新します。
	UpdateSortKeys(ctx context.Context, userID uint, codeOrder []string) error
	// AddBatch はユーザーのウォッチリストに複数の銘柄を一括追加します（デフォルト銘柄用）。
	AddBatch(ctx context.Context, userSymbols []entity.UserSymbol) error
	// MaxSortKey はユーザーのウォッチリスト内の最大sort_keyを返します。
	MaxSortKey(ctx context.Context, userID uint) (int, error)
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
// sort_keyは既存の最大値+10に自動設定されます。
func (u *WatchlistUsecase) AddSymbol(ctx context.Context, userID uint, symbolCode string) error {
	if symbolCode == "" {
		return fmt.Errorf("symbol code must not be empty")
	}

	maxKey, err := u.repo.MaxSortKey(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get max sort key: %w", err)
	}

	us := &entity.UserSymbol{
		UserID:     userID,
		SymbolCode: symbolCode,
		SortKey:    maxKey + 10,
	}
	return u.repo.Add(ctx, us)
}

// RemoveSymbol はユーザーのウォッチリストから銘柄を削除します。
func (u *WatchlistUsecase) RemoveSymbol(ctx context.Context, userID uint, symbolCode string) error {
	return u.repo.Remove(ctx, userID, symbolCode)
}

// ReorderSymbols はユーザーのウォッチリスト銘柄の並び順を更新します。
// codeOrderの順番に従い、sort_key = index * 10 で設定します。
func (u *WatchlistUsecase) ReorderSymbols(ctx context.Context, userID uint, codeOrder []string) error {
	if len(codeOrder) == 0 {
		return fmt.Errorf("code order must not be empty")
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
