// Package handler provides HTTP handlers for platform-level endpoints.
package handler

import "github.com/gin-gonic/gin"

// Health handles the /healthz endpoint for service health checks.
// It responds appropriately based on the HTTP method and prevents caching.
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
