// Package middleware はプラットフォーム共通のHTTPミドルウェアを提供します。
package middleware

import "net/http"

// SecurityHeaders はセキュリティ関連のHTTPレスポンスヘッダーを設定するミドルウェアを返します。
// このAPIサーバーはJSONのみを返すため、CSPは最も制限的な設定を使用します。
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Content-Security-Policy", "default-src 'none'")
			next.ServeHTTP(w, r)
		})
	}
}
