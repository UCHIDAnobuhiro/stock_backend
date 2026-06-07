package main

import (
	"log/slog"
	"os"

	"github.com/UCHIDAnobuhiro/stock-backend/internal/app/migrate"
)

// main はロガーを設定し、migrate.Run の戻り値で os.Exit するだけの薄いラッパー。
// os.Exit は defer を実行しないため、後処理が走るよう実体は internal/app/migrate に分離している。
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	os.Exit(migrate.Run(os.Args[1:]))
}
