package handler

import "github.com/gin-gonic/gin"

func Health(c *gin.Context) {
	// Explicitly prevent caching
	c.Header("Cache-Control", "no-store")

	// Return 200 or 204 for all GET/HEAD/OPTIONS requests
	switch c.Request.Method {
	case "HEAD":
		c.Status(200)
	case "OPTIONS":
		c.Status(204)
	default:
		c.JSON(200, gin.H{"status": "ok"})
	}
}
