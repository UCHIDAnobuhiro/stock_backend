package di

import (
	"context"

	candlesusecase "stock_backend/internal/feature/candles/usecase"
	"stock_backend/internal/feature/symbollist/domain/entity"
)

// SymbolLister は symbollist リポジトリが提供するアクティブ銘柄取得インターフェースです。
// 直接 *symbolRepository に依存せず、symbollist フィーチャーから ingest 側へのデータ受け渡しを抽象化します。
type SymbolLister interface {
	ListActive(ctx context.Context) ([]entity.Symbol, error)
}

// ingestSymbolAdapter は symbollist の Symbol を candlesusecase.ActiveSymbol へ詰め替えます。
// feature 同士の直接依存を避けるため DI 層で変換を行います。
type ingestSymbolAdapter struct {
	src SymbolLister
}

// NewIngestSymbolAdapter は ingest 用の SymbolRepository 実装を返します。
func NewIngestSymbolAdapter(src SymbolLister) candlesusecase.SymbolRepository {
	return &ingestSymbolAdapter{src: src}
}

// ListActiveSymbols はアクティブな全銘柄をコード+タイムゾーンの組として返します。
func (a *ingestSymbolAdapter) ListActiveSymbols(ctx context.Context) ([]candlesusecase.ActiveSymbol, error) {
	syms, err := a.src.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]candlesusecase.ActiveSymbol, 0, len(syms))
	for _, s := range syms {
		out = append(out, candlesusecase.ActiveSymbol{Code: s.Code, Timezone: s.Timezone})
	}
	return out, nil
}
