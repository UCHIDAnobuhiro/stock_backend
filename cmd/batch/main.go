package main

import (
	"os"

	"stock_backend/internal/app/batch"
)

// main は batch.Run の戻り値で os.Exit するだけの薄いラッパー。
// os.Exit は defer を実行しないため、後処理が走るよう実体は internal/app/batch に分離している。
func main() {
	os.Exit(batch.Run(os.Args[1:]))
}
