// cmd/migrate はスキーマのマイグレーションだけを実行するバイナリ。
//
// cmd/server も RUN_MIGRATIONS=true でマイグレーションを実行できるが、
// サーバー本体は GCP (Vision / Gemini) クライアント初期化も伴うため、
// 認証情報のない CI 環境や Cloud Run の pre-deploy ジョブでは実行できない。
// このバイナリは DB 接続と AutoMigrate + FK 制約追加のみを行って終了する。
package main

import (
	"log/slog"
	"os"

	authadapters "stock_backend/internal/feature/auth/adapters"
	authentity "stock_backend/internal/feature/auth/domain/entity"
	candlesadapters "stock_backend/internal/feature/candles/adapters"
	symbolentity "stock_backend/internal/feature/symbollist/domain/entity"
	watchlistadapters "stock_backend/internal/feature/watchlist/adapters"
	watchlistentity "stock_backend/internal/feature/watchlist/domain/entity"
	infradb "stock_backend/internal/platform/db"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	db := infradb.OpenDB()

	if err := infradb.RunMigrations(db,
		&authentity.User{},
		&authentity.OAuthAccount{},
		&candlesadapters.CandleModel{},
		&symbolentity.Symbol{},
		&watchlistentity.UserSymbol{},
	); err != nil {
		slog.Error("failed to migrate", "error", err)
		os.Exit(1)
	}
	if err := authadapters.MakePasswordNullable(db); err != nil {
		slog.Error("failed to make password nullable", "error", err)
		os.Exit(1)
	}
	if err := authadapters.AddOAuthAccountsFKConstraints(db); err != nil {
		slog.Error("failed to add oauth_accounts FK constraints", "error", err)
		os.Exit(1)
	}
	if err := watchlistadapters.AddFKConstraints(db); err != nil {
		slog.Error("failed to add watchlist FK constraints", "error", err)
		os.Exit(1)
	}
	if err := candlesadapters.AddFKConstraints(db); err != nil {
		slog.Error("failed to add candles FK constraints", "error", err)
		os.Exit(1)
	}

	slog.Info("migrate ok")
}
