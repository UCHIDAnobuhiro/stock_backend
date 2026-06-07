package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AccessLog は各 HTTP リクエストを slog の構造化ログとして出力する Gin ミドルウェアを返します。
// Gin 標準のテキストアクセスログ（gin.Logger）の代わりに使用し、Cloud Logging が解釈できる
// httpRequest フィールドとトレース相関フィールドを出力します。
//
// projectID（GOOGLE_CLOUD_PROJECT）が指定されている場合、X-Cloud-Trace-Context ヘッダーから
// トレース ID を抽出してログをリクエスト単位で相関させます。空の場合はトレースフィールドを
// 出力しません。
func AccessLog(projectID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path = path + "?" + raw
		}

		c.Next()

		status := c.Writer.Status()
		latency := time.Since(start)

		responseSize := c.Writer.Size()
		if responseSize < 0 {
			responseSize = 0
		}

		// Cloud Logging の httpRequest 構造化フィールド。
		// latency は protobuf Duration の JSON 表現（秒 + "s"）、サイズ系は文字列で表す。
		attrs := []slog.Attr{
			slog.Any("httpRequest", map[string]any{
				"requestMethod": c.Request.Method,
				"requestUrl":    path,
				"status":        status,
				"responseSize":  fmt.Sprintf("%d", responseSize),
				"userAgent":     c.Request.UserAgent(),
				"remoteIp":      c.ClientIP(),
				"protocol":      c.Request.Proto,
				"latency":       fmt.Sprintf("%.9fs", latency.Seconds()),
			}),
		}

		if trace, span, ok := traceContext(projectID, c.Request.Header.Get("X-Cloud-Trace-Context")); ok {
			attrs = append(attrs,
				slog.String("logging.googleapis.com/trace", trace),
				slog.String("logging.googleapis.com/spanId", span),
			)
		}

		// Gin が収集したハンドラー内エラーがあれば付加する。
		if msg := strings.TrimSpace(c.Errors.ByType(gin.ErrorTypePrivate).String()); msg != "" {
			attrs = append(attrs, slog.String("error", msg))
		}

		slog.LogAttrs(c.Request.Context(), severityForStatus(status), "request", attrs...)
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
