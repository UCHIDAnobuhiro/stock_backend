// Package usecase はwatchlistフィーチャーのビジネスロジックを実装します。
package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"stock_backend/internal/feature/watchlist/domain/entity"
)

// WatchlistRepository はウォッチリスト操作の永続化層を抽象化します。
type WatchlistRepository interface {
	ListByUser(ctx context.Context, userID uint) ([]entity.UserSymbol, error)
	// Add はsort_keyを指定してウォッチリストに銘柄を追加します。
	Add(ctx context.Context, entry entity.UserSymbol) error
	// AddWithNextSortKey はsort_keyをトランザクション内でMAX+1採番して銘柄を追加します。
	// MaxSortKey取得とInsertをアトミックに実行するため、並行追加時の重複順位を防ぎます。
	AddWithNextSortKey(ctx context.Context, userID uint, symbolCode string) error
	Remove(ctx context.Context, userID uint, symbolCode string) error
	UpdateSortKeys(ctx context.Context, userID uint, entries []entity.UserSymbol) error
}

// SymbolExistsChecker は銘柄の存在確認を行うインターフェースです。
// watchlist usecase が symbollist feature に直接依存しないよう、
// 最小限の読み取り専用インターフェースをここで定義します。
type SymbolExistsChecker interface {
	Exists(ctx context.Context, code string) (bool, error)
}

// WatchlistUsecase はウォッチリスト操作のビジネスロジックを提供します。
type WatchlistUsecase struct {
	repo          WatchlistRepository
	symbolChecker SymbolExistsChecker
}

// NewWatchlistUsecase は指定されたリポジトリと銘柄チェッカーで WatchlistUsecase の新しいインスタンスを生成します。
func NewWatchlistUsecase(repo WatchlistRepository, symbolChecker SymbolExistsChecker) *WatchlistUsecase {
	return &WatchlistUsecase{repo: repo, symbolChecker: symbolChecker}
}

// ListUserSymbols はユーザーのウォッチリストをソート順で返します。
func (u *WatchlistUsecase) ListUserSymbols(ctx context.Context, userID uint) ([]entity.UserSymbol, error) {
	return u.repo.ListByUser(ctx, userID)
}

// AddSymbol はウォッチリストに銘柄を追加します。
// symbols テーブルに存在しない銘柄コードの場合は ErrSymbolNotFound を返します。
// 既にウォッチリストに存在する場合は ErrAlreadyInWatchlist を返します。
func (u *WatchlistUsecase) AddSymbol(ctx context.Context, userID uint, symbolCode string) error {
	exists, err := u.symbolChecker.Exists(ctx, symbolCode)
	if err != nil {
		return fmt.Errorf("checking symbol existence: %w", err)
	}
	if !exists {
		return ErrSymbolNotFound
	}

	return u.repo.AddWithNextSortKey(ctx, userID, symbolCode)
}

// RemoveSymbol はウォッチリストから銘柄を削除します。
func (u *WatchlistUsecase) RemoveSymbol(ctx context.Context, userID uint, symbolCode string) error {
	return u.repo.Remove(ctx, userID, symbolCode)
}

// ReorderSymbols はウォッチリストの並び順を更新します。
func (u *WatchlistUsecase) ReorderSymbols(ctx context.Context, userID uint, orderedCodes []string) error {
	entries := make([]entity.UserSymbol, 0, len(orderedCodes))
	for i, code := range orderedCodes {
		entries = append(entries, entity.UserSymbol{
			UserID:     userID,
			SymbolCode: code,
			SortKey:    i,
		})
	}
	return u.repo.UpdateSortKeys(ctx, userID, entries)
}

// OnUserCreated は PostSignupHook インターフェースを実装します。
// サインアップ直後に呼ばれ、デフォルト銘柄を追加します。
func (u *WatchlistUsecase) OnUserCreated(ctx context.Context, userID uint) error {
	return u.InitializeDefaults(ctx, userID)
}

// InitializeDefaults は新規ユーザー向けにデフォルト銘柄（AAPL/MSFT/GOOGL）を追加します。
// symbols テーブルに存在する銘柄のみ追加します（存在しない場合はスキップ）。
func (u *WatchlistUsecase) InitializeDefaults(ctx context.Context, userID uint) error {
	defaultSymbols := []string{"AAPL", "MSFT", "GOOGL"}

	for i, code := range defaultSymbols {
		exists, err := u.symbolChecker.Exists(ctx, code)
		if err != nil {
			return fmt.Errorf("checking symbol %s: %w", code, err)
		}
		if !exists {
			slog.Warn("default symbol not found in symbols table, skipping", "code", code)
			continue
		}
		if err := u.repo.Add(ctx, entity.UserSymbol{
			UserID:     userID,
			SymbolCode: code,
			SortKey:    i,
		}); err != nil {
			return fmt.Errorf("adding default symbol %s: %w", code, err)
		}
	}
	return nil
}
