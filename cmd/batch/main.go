package main

import (
	"log/slog"
	"os"

	"github.com/UCHIDAnobuhiro/stock-backend/internal/app/batch"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/app/config"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/infra/logging"
)

// main は設定を読み込んでロガーを設定し、batch.Run の戻り値で os.Exit するだけの薄いラッパー。
// os.Exit は defer を実行しないため、後処理が走るよう実体は internal/app/batch に分離している。
func main() {
	cfg, err := config.LoadBatch()
	logger := slog.New(logging.NewHandler(os.Stdout, cfg.Log.Level, cfg.Log.UseJSON))
	slog.SetDefault(logger)
	for _, w := range cfg.Warnings {
		slog.Warn(w)
	}
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(2)
	}

	os.Exit(batch.Run(cfg, os.Args[1:]))
}
