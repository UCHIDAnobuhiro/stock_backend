// Package router configures HTTP routes for the application.
package router

import (
	authhandler "stock_backend/internal/feature/auth/transport/handler"
	candleshandler "stock_backend/internal/feature/candles/transport/handler"
	symbollisthandler "stock_backend/internal/feature/symbollist/transport/handler"
	handler "stock_backend/internal/platform/http/handler"
	jwtmw "stock_backend/internal/platform/jwt"

	"github.com/gin-gonic/gin"
)

// NewRouter creates and configures a Gin router with all application routes.
// It sets up public routes (signup, login) and protected routes (candles, symbols)
// with JWT authentication middleware.
func NewRouter(authHandler *authhandler.AuthHandler, candles *candleshandler.CandlesHandler,
	symbol *symbollisthandler.SymbolHandler) *gin.Engine {
	r := gin.Default()

	// Health check endpoint (unversioned)
	r.GET("/healthz", handler.Health)

	// API v1 routes
	v1 := r.Group("/v1")
	{
		// Public routes (no authentication required)
		v1.POST("/signup", authHandler.Signup)
		v1.POST("/login", authHandler.Login)

		// Protected routes (authentication required)
		auth := v1.Group("/")
		auth.Use(jwtmw.AuthRequired())
		{
			auth.GET("/candles/:code", candles.GetCandlesHandler)
			auth.GET("/symbols", symbol.List)
		}
	}

	return r
}
