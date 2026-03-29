// Package usecase はwatchlistフィーチャーのビジネスロジックを実装します。
package usecase

import "errors"

var (
	// ErrSymbolNotFound は指定された銘柄コードが symbols テーブルに存在しない場合のエラーです。
	ErrSymbolNotFound = errors.New("symbol not found")

	// ErrAlreadyInWatchlist は銘柄が既にウォッチリストに存在する場合のエラーです。
	ErrAlreadyInWatchlist = errors.New("symbol already in watchlist")

	// ErrNotInWatchlist は削除対象の銘柄がウォッチリストに存在しない場合のエラーです。
	ErrNotInWatchlist = errors.New("symbol not in watchlist")
)
