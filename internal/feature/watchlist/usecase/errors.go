// Package usecase はwatchlistフィーチャーのビジネスロジックを実装します。
package usecase

import "errors"

var (
	// ErrSymbolAlreadyExists はウォッチリストに既に同じ銘柄が存在する場合のエラーです。
	ErrSymbolAlreadyExists = errors.New("symbol already exists in watchlist")
	// ErrSymbolNotFound はウォッチリストに指定された銘柄が存在しない場合のエラーです。
	ErrSymbolNotFound = errors.New("symbol not found in watchlist")
)
