package main

import (
	"log/slog"
	"os"

	"stock_backend/internal/app/batch"
)

// main はロガーを設定し、batch.Run の戻り値で os.Exit するだけの薄いラッパー。
// os.Exit は defer を実行しないため、後処理が走るよう実体は internal/app/batch に分離している。
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	os.Exit(batch.Run(os.Args[1:]))
}
