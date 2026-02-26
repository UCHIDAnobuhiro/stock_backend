package ratelimit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain は全テスト共通のテスト環境を設定します。
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

// TestByIP_Allowed はレートリミット内のリクエストがハンドラーまで到達し200を返すことを検証します。
func TestByIP_Allowed(t *testing.T) {
	t.Parallel()

	rdb, mock := redismock.NewClientMock()
	defer func() { _ = rdb.Close() }()

	window := time.Minute
	setupPipelineMock(mock, "rl:test:ip:192.0.2.1", 0, window)

	limiter := NewLimiter(rdb)
	cfg := IPRateLimitConfig{Prefix: "rl:test:ip", Limit: 10, Window: window}

	r := gin.New()
	r.POST("/test", ByIP(limiter, cfg), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestByIP_RateLimited はレートリミット超過時に429とRetry-Afterヘッダーが返され、
// ハンドラーが呼ばれないことを検証します。
func TestByIP_RateLimited(t *testing.T) {
	t.Parallel()

	rdb, mock := redismock.NewClientMock()
	defer func() { _ = rdb.Close() }()

	window := time.Minute
	setupPipelineMock(mock, "rl:test:ip:192.0.2.1", 10, window) // at limit

	limiter := NewLimiter(rdb)
	cfg := IPRateLimitConfig{Prefix: "rl:test:ip", Limit: 10, Window: window}

	handlerCalled := false
	r := gin.New()
	r.POST("/test", ByIP(limiter, cfg), func(c *gin.Context) {
		handlerCalled = true
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.False(t, handlerCalled, "ハンドラーは呼ばれるべきではない")

	// Retry-Afterヘッダーの検証
	assert.Equal(t, "60", w.Header().Get("Retry-After"))

	// レスポンスボディの検証
	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "too many requests", body["error"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestByIP_NilRedis_Allowed はRedisクライアントがnilの場合にミドルウェアがリクエストを通過させることを検証します。
func TestByIP_NilRedis_Allowed(t *testing.T) {
	t.Parallel()

	limiter := NewLimiter(nil)
	cfg := IPRateLimitConfig{Prefix: "rl:test:ip", Limit: 10, Window: time.Minute}

	r := gin.New()
	r.POST("/test", ByIP(limiter, cfg), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
