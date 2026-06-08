package main

import (
	"log/slog"
	"os"

	"github.com/UCHIDAnobuhiro/stock-backend/internal/app/batch"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/app/config"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/infra/logging"
)

// main はロガーを設定し、batch.Run の戻り値で os.Exit するだけの薄いラッパー。
// os.Exit は defer を実行しないため、後処理が走るよう実体は internal/app/batch に分離している。
func main() {
	useJSON, _ := config.ParseLogFormat(os.Getenv("LOG_FORMAT"), os.Getenv("APP_ENV"))
	logger := slog.New(logging.NewHandler(os.Stdout, slog.LevelInfo, useJSON))
	slog.SetDefault(logger)

	os.Exit(batch.Run(os.Args[1:]))
}
