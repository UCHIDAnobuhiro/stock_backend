package usecase

import (
	"context"
	"log/slog"
	"time"

	"stock_backend/internal/feature/symbollist/domain/entity"
	"stock_backend/internal/shared/ratelimiter"
)

// LogoProvider は外部APIからロゴURLを取得するリポジトリを抽象化します。
type LogoProvider interface {
	GetLogoURL(ctx context.Context, symbol string) (string, error)
}

// LogoSymbolRepository はロゴURL取得バッチで使う銘柄リポジトリを抽象化します。
type LogoSymbolRepository interface {
	ListActive(ctx context.Context) ([]entity.Symbol, error)
	UpdateLogoURL(ctx context.Context, code, logoURL string, updatedAt time.Time) error
}

// LogoIngestResult はロゴURL取得バッチの銘柄単位の集計結果を表します。
type LogoIngestResult struct {
	Total     int
	Succeeded int
	Failed    int
}

// FailureRate は失敗率を [0.0, 1.0] で返します。Total が 0 の場合は 0 を返します。
func (r LogoIngestResult) FailureRate() float64 {
	if r.Total == 0 {
		return 0
	}
	return float64(r.Failed) / float64(r.Total)
}

// LogoIngestUsecase はactive銘柄のロゴURLを外部APIから取得して保存します。
type LogoIngestUsecase struct {
	logoProvider LogoProvider
	symbolRepo   LogoSymbolRepository
	rateLimiter  ratelimiter.RateLimiterInterface
	now          func() time.Time
}

// NewLogoIngestUsecase はLogoIngestUsecaseの新しいインスタンスを生成します。
func NewLogoIngestUsecase(provider LogoProvider, symbolRepo LogoSymbolRepository, rateLimiter ratelimiter.RateLimiterInterface) *LogoIngestUsecase {
	return &LogoIngestUsecase{
		logoProvider: provider,
		symbolRepo:   symbolRepo,
		rateLimiter:  rateLimiter,
		now:          time.Now,
	}
}

// IngestAll はactive銘柄のロゴURLを毎回再取得し、成功時のみDBを更新します。
// 銘柄単位の失敗では処理を止めず、既存logo_urlも保持します。
func (u *LogoIngestUsecase) IngestAll(ctx context.Context) (LogoIngestResult, error) {
	symbols, err := u.symbolRepo.ListActive(ctx)
	if err != nil {
		return LogoIngestResult{}, err
	}

	result := LogoIngestResult{Total: len(symbols)}
	for _, s := range symbols {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if err := u.rateLimiter.WaitIfNeeded(ctx); err != nil {
			return result, err
		}

		logoURL, err := u.logoProvider.GetLogoURL(ctx, s.Code)
		if err != nil {
			slog.Error("failed to fetch logo url", "symbol", s.Code, "error", err)
			result.Failed++
			continue
		}
		if err := u.symbolRepo.UpdateLogoURL(ctx, s.Code, logoURL, u.now()); err != nil {
			slog.Error("failed to update logo url", "symbol", s.Code, "error", err)
			result.Failed++
			continue
		}
		result.Succeeded++
	}
	return result, nil
}
