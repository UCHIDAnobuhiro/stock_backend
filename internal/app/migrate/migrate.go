// Package migrate はスキーママイグレーションの実行ロジックを提供します。
//
// 本番デプロイでは Cloud Run pre-deploy ジョブ等で cmd/migrate を起動して
// goose によるマイグレーションを適用し、その後 cmd/api をデプロイします。
//
// 使い方:
//
//	migrate [up|down|down-to|status|version|reset|redo|up-by-one|up-to] [arg]
//
// 引数なしで実行した場合は up を適用します。
package migrate

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/UCHIDAnobuhiro/stock-backend/internal/app/config"
	infradb "github.com/UCHIDAnobuhiro/stock-backend/internal/infra/db"
)

// allowedCommands は本バイナリから実行を許容する goose サブコマンドです。
// create / fix のような開発者ローカル専用の操作は除外し、デプロイで使うものに限定します。
var allowedCommands = map[string]struct{}{
	"up":        {},
	"up-by-one": {},
	"up-to":     {},
	"down":      {},
	"down-to":   {},
	"redo":      {},
	"reset":     {},
	"status":    {},
	"version":   {},
}

// Run は goose サブコマンド（コマンド引数）に応じてマイグレーションを実行し、終了コードを返す。
// 引数なしの場合は up を適用する。環境変数から読み込んだ設定は cfg として注入される。
// os.Exit は呼ばず、終了コードを返すのみ（呼び出し側の main で os.Exit する）。
func Run(cfg *config.Config, args []string) int {
	cmd := "up"
	var extra []string
	if len(args) > 0 {
		cmd = args[0]
		extra = args[1:]
	}
	if _, ok := allowedCommands[cmd]; !ok {
		slog.Error("unsupported goose command", "command", cmd, "allowed", slices.Sorted(maps.Keys(allowedCommands)))
		return 2
	}

	db, err := infradb.OpenSQL(cfg.DB)
	if err != nil {
		slog.Error("DB open failed", "error", err)
		return 1
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Warn("failed to close DB", "error", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := infradb.RunGoose(ctx, db, cmd, extra...); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Error("migration timed out", "error", err)
		} else {
			slog.Error("migration failed", "command", cmd, "error", err)
		}
		return 1
	}
	slog.Info("migration ok", "command", cmd)
	return 0
}
