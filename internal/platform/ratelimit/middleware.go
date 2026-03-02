package ratelimit

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"stock_backend/internal/api"
)

// IPRateLimitConfig はIPベースのレートリミットの設定を保持します。
type IPRateLimitConfig struct {
	Prefix string        // Redisキーのプレフィックス（例: "rl:login:ip"）
	Limit  int           // ウィンドウ内の最大リクエスト数
	Window time.Duration // スライディングウィンドウの時間幅
}

// ByIP はIPアドレスベースのレートリミットGinミドルウェアを返します。
func ByIP(limiter *Limiter, cfg IPRateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("%s:%s", cfg.Prefix, c.ClientIP())
		result := limiter.Allow(c.Request.Context(), key, cfg.Limit, cfg.Window)

		if !result.Allowed {
			slog.Warn("rate limit exceeded",
				"type", "ip",
				"ip", c.ClientIP(),
				"prefix", cfg.Prefix,
			)
			c.Header("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, api.ErrorResponse{
				Error: "too many requests",
			})
			return
		}
		c.Next()
	}
}
