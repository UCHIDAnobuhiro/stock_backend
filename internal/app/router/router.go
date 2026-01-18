package router

import (
	authhandler "stock_backend/internal/feature/auth/transport/handler"
	candleshandler "stock_backend/internal/feature/candles/transport/handler"
	symbollisthandler "stock_backend/internal/feature/symbollist/transport/handler"
	handler "stock_backend/internal/platform/http/handler"
	jwtmw "stock_backend/internal/platform/jwt"

	"github.com/gin-gonic/gin"
)

func NewRouter(authHandler *authhandler.AuthHandler, candles *candleshandler.CandlesHandler,
	symbol *symbollisthandler.SymbolHandler) *gin.Engine {
	r := gin.Default()

	// Public routes (no authentication required)
	// Health check endpoint
	r.GET("/healthz", handler.Health)
	// New user registration
	r.POST("/signup", authHandler.Signup)
	// Login (JWT issuance)
	r.POST("/login", authHandler.Login)

	// Protected routes (authentication required)
	// Create route group with r.Group("/")
	auth := r.Group("/")
	// Apply jwtmw.AuthRequired() middleware
	// This requires JWT in the request header
	auth.Use(jwtmw.AuthRequired())
	{
		auth.GET("/candles/:code", candles.GetCandlesHandler)
		auth.GET("/symbols", symbol.List)
	}

	return r
}
