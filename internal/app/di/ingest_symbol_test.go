package di

import (
	"context"
	"errors"
	"testing"

	candlesusecase "stock_backend/internal/feature/candles/usecase"
	"stock_backend/internal/feature/symbollist/domain/entity"
)

type stubSymbolLister struct {
	syms []entity.Symbol
	err  error
}

func (s *stubSymbolLister) ListActive(ctx context.Context) ([]entity.Symbol, error) {
	return s.syms, s.err
}

func TestIngestSymbolAdapter_ListActiveSymbols(t *testing.T) {
	t.Parallel()

	stub := &stubSymbolLister{
		syms: []entity.Symbol{
			{Code: "AAPL", Timezone: "America/New_York"},
			{Code: "7203.T", Timezone: "Asia/Tokyo"},
		},
	}

	got, err := NewIngestSymbolAdapter(stub).ListActiveSymbols(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []candlesusecase.ActiveSymbol{
		{Code: "AAPL", Timezone: "America/New_York"},
		{Code: "7203.T", Timezone: "Asia/Tokyo"},
	}
	if len(got) != len(want) {
		t.Fatalf("len: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d]: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestIngestSymbolAdapter_ListActiveSymbols_PropagatesError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("db down")
	stub := &stubSymbolLister{err: wantErr}

	got, err := NewIngestSymbolAdapter(stub).ListActiveSymbols(context.Background())
	if !errors.Is(err, wantErr) {
		t.Errorf("err: got %v, want %v", err, wantErr)
	}
	if got != nil {
		t.Errorf("got %v, want nil on error", got)
	}
}

func TestIngestSymbolAdapter_ListActiveSymbols_Empty(t *testing.T) {
	t.Parallel()

	stub := &stubSymbolLister{syms: []entity.Symbol{}}

	got, err := NewIngestSymbolAdapter(stub).ListActiveSymbols(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got len %d, want 0", len(got))
	}
}
