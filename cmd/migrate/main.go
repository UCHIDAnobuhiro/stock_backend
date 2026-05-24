// cmd/migrate はスキーママイグレーションだけを実行するバイナリです。
//
// 本番デプロイでは Cloud Run pre-deploy ジョブ等で本バイナリを起動して
// goose によるマイグレーションを適用し、その後 cmd/server をデプロイします。
//
// 使い方:
//
//	migrate [up|down|down-to|status|version|reset|redo|up-by-one|up-to] [arg]
//
// 引数なしで実行した場合は up を適用します。
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	infradb "stock_backend/internal/platform/db"
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

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cmd := "up"
	var extra []string
	if len(args) > 0 {
		cmd = args[0]
		extra = args[1:]
	}
	if _, ok := allowedCommands[cmd]; !ok {
		slog.Error("unsupported goose command", "command", cmd, "allowed", keys(allowedCommands))
		return 2
	}

	db := infradb.OpenSQL()
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

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
