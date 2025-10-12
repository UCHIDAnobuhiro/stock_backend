package handler

import "github.com/gin-gonic/gin"

func Health(c *gin.Context) {
	// キャッシュされないように明示
	c.Header("Cache-Control", "no-store")

	// GET/HEAD/OPTIONS すべて 200 or 204 で返す
	switch c.Request.Method {
	case "HEAD":
		c.Status(200)
	case "OPTIONS":
		c.Status(204)
	default:
		c.JSON(200, gin.H{"status": "ok"})
	}
}
