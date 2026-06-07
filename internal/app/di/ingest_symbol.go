package di

import (
	"context"

	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/candles"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/symbollist"
)

// SymbolLister は symbollist リポジトリが提供するアクティブ銘柄取得インターフェースです。
// 直接 *repository に依存せず、symbollist フィーチャーから ingest 側へのデータ受け渡しを抽象化します。
type SymbolLister interface {
	ListActive(ctx context.Context) ([]symbollist.Symbol, error)
}

// ingestSymbolAdapter は symbollist の Symbol を candles.ActiveSymbol へ詰め替えます。
// feature 同士の直接依存を避けるため DI 層で変換を行います。
type ingestSymbolAdapter struct {
	src SymbolLister
}

// NewIngestSymbolAdapter は ingest 用の SymbolRepository 実装を返します。
func NewIngestSymbolAdapter(src SymbolLister) candles.SymbolRepository {
	return &ingestSymbolAdapter{src: src}
}

// ListActiveSymbols はアクティブな全銘柄をコード+タイムゾーンの組として返します。
func (a *ingestSymbolAdapter) ListActiveSymbols(ctx context.Context) ([]candles.ActiveSymbol, error) {
	syms, err := a.src.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]candles.ActiveSymbol, 0, len(syms))
	for _, s := range syms {
		out = append(out, candles.ActiveSymbol{Code: s.Code, Timezone: s.Timezone})
	}
	return out, nil
}
