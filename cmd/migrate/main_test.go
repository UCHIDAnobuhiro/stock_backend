package main

import (
	"io"
	"log/slog"
	"testing"
)

// TestRun_RejectsUnsupportedCommand は allowedCommands に含まれないサブコマンドを
// run() が拒否し、終了コード 2 を返すことを検証します。
// OpenSQL を呼ぶ前に弾かれるため、DB なしで実行可能です。
func TestRun_RejectsUnsupportedCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "create (developer-only)", args: []string{"create", "foo", "sql"}},
		{name: "fix (developer-only)", args: []string{"fix"}},
		{name: "unknown command", args: []string{"no-such"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := run(tt.args)
			if got != 2 {
				t.Errorf("run(%v) = %d, want 2", tt.args, got)
			}
		})
	}
}

func TestRun_DoesNotChangeDefaultLogger(t *testing.T) {
	original := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(original)
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	slog.SetDefault(logger)

	if got := run([]string{"no-such"}); got != 2 {
		t.Fatalf("run() = %d, want 2", got)
	}
	if got := slog.Default(); got != logger {
		t.Errorf("slog.Default() = %p, want %p", got, logger)
	}
}

// TestAllowedCommands_ContainsExpectedSet は本バイナリが受け付けるべき
// サブコマンドの集合を固定化します（誤って広げない / 狭めないため）。
func TestAllowedCommands_ContainsExpectedSet(t *testing.T) {
	t.Parallel()

	want := []string{
		"up", "up-by-one", "up-to",
		"down", "down-to",
		"redo", "reset",
		"status", "version",
	}
	if len(allowedCommands) != len(want) {
		t.Errorf("allowedCommands size = %d, want %d", len(allowedCommands), len(want))
	}
	for _, c := range want {
		if _, ok := allowedCommands[c]; !ok {
			t.Errorf("allowedCommands missing %q", c)
		}
	}

	// 開発者ローカル専用コマンドが含まれていないことを検証
	for _, banned := range []string{"create", "fix"} {
		if _, ok := allowedCommands[banned]; ok {
			t.Errorf("allowedCommands must not include %q (developer-only)", banned)
		}
	}
}
