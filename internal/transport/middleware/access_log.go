package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/UCHIDAnobuhiro/stock-backend/internal/transport/httpx"
)

// AccessLog は各 HTTP リクエストを slog の構造化ログとして出力するミドルウェアを返します。
// Cloud Logging が解釈できる httpRequest フィールドとトレース相関フィールドを出力します。
//
// projectID（GOOGLE_CLOUD_PROJECT）が指定されている場合、X-Cloud-Trace-Context ヘッダーから
// トレース ID を抽出してログをリクエスト単位で相関させます。空の場合はトレースフィールドを
// 出力しません。
func AccessLog(projectID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			path := r.URL.Path
			if raw := r.URL.RawQuery; raw != "" {
				path = path + "?" + raw
			}

			// ステータスコードと書き込みバイト数を捕捉するためレスポンスライターをラップする。
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			status := ww.Status()
			if status == 0 {
				// ハンドラーが WriteHeader を呼ばなかった場合、net/http は 200 を返す。
				status = http.StatusOK
			}
			latency := time.Since(start)
			responseSize := ww.BytesWritten()

			// Cloud Logging の httpRequest 構造化フィールド。
			// latency は protobuf Duration の JSON 表現（秒 + "s"）、サイズ系は文字列で表す。
			attrs := []slog.Attr{
				slog.Any("httpRequest", map[string]any{
					"requestMethod": r.Method,
					"requestUrl":    path,
					"status":        status,
					"responseSize":  fmt.Sprintf("%d", responseSize),
					"userAgent":     r.UserAgent(),
					"remoteIp":      httpx.ClientIP(r),
					"protocol":      r.Proto,
					"latency":       fmt.Sprintf("%.9fs", latency.Seconds()),
				}),
			}

			if trace, span, ok := traceContext(projectID, r.Header.Get("X-Cloud-Trace-Context")); ok {
				attrs = append(attrs,
					slog.String("logging.googleapis.com/trace", trace),
					slog.String("logging.googleapis.com/spanId", span),
				)
			}

			slog.LogAttrs(r.Context(), severityForStatus(status), "request", attrs...)
		})
	}
}

// severityForStatus は HTTP ステータスコードを slog のレベルに対応付けます。
// 5xx は Error、4xx は Warn、それ以外は Info とします。
func severityForStatus(status int) slog.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case status >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

// traceContext は X-Cloud-Trace-Context ヘッダー（"TRACE_ID/SPAN_ID;o=1" 形式）を
// パースし、Cloud Logging が要求する trace リソース名と span ID を返します。
// projectID が空、またはヘッダーから trace ID を取得できない場合は ok=false を返します。
func traceContext(projectID, header string) (trace, span string, ok bool) {
	if projectID == "" || header == "" {
		return "", "", false
	}

	traceID := header
	if i := strings.IndexByte(traceID, '/'); i >= 0 {
		span = traceID[i+1:]
		traceID = traceID[:i]
		if j := strings.IndexByte(span, ';'); j >= 0 {
			span = span[:j]
		}
	}
	if traceID == "" {
		return "", "", false
	}

	return fmt.Sprintf("projects/%s/traces/%s", projectID, traceID), span, true
}
