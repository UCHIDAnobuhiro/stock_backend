// Package config はサーバー起動時に環境変数から設定を読み取るための
// 純粋関数ヘルパーを提供します。
package config

import (
	"strconv"
	"strings"
)

// ParseCORSOrigins は CORS_ALLOWED_ORIGINS env の生文字列を、カンマ区切りで
// trim して空要素を除いたスライスに変換する。raw が空なら nil を返し、
// 呼び出し側にデフォルト適用を委ねる。
func ParseCORSOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	if len(origins) == 0 {
		return nil
	}
	return origins
}

// ParseBoolString は raw を bool として解釈する。
//   - raw が空文字の場合は (fallback, true) を返す（未設定は正常系扱い）。
//   - strconv.ParseBool で解釈できる場合は (parsed, true) を返す。
//   - 不正値の場合は (fallback, false) を返す。呼び出し側で警告ログなどの判断に利用する。
//
// env を直接読まず純粋な文字列を受け取るため、呼び出し側は os.Getenv 等で取得した値を渡す。
func ParseBoolString(raw string, fallback bool) (value bool, ok bool) {
	if raw == "" {
		return fallback, true
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback, false
	}
	return parsed, true
}
