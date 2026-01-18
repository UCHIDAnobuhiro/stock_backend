// Package handler provides HTTP handlers for the symbollist feature.
package handler

import (
	"context"
	"net/http"
	"stock_backend/internal/feature/symbollist/domain/entity"
	"stock_backend/internal/feature/symbollist/transport/http/dto"

	"github.com/gin-gonic/gin"
)

// SymbolUsecase defines the use case interface for symbol (stock ticker) operations.
// Following Go convention: interfaces are defined by the consumer (handler), not the provider (usecase).
type SymbolUsecase interface {
	ListActiveSymbols(ctx context.Context) ([]entity.Symbol, error)
}

// SymbolHandler handles HTTP requests related to symbol information.
type SymbolHandler struct {
	uc SymbolUsecase
}

// NewSymbolHandler creates a new SymbolHandler.
func NewSymbolHandler(uc SymbolUsecase) *SymbolHandler {
	return &SymbolHandler{uc: uc}
}

// List retrieves the list of active symbols.
// It calls the usecase to fetch the symbol list, converts it to DTOs,
// and returns them as a JSON response.
// Returns 500 Internal Server Error if the usecase returns an error.
func (h *SymbolHandler) List(c *gin.Context) {
	symbols, err := h.uc.ListActiveSymbols(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]dto.SymbolItem, 0, len(symbols))
	for _, s := range symbols {
		out = append(out, dto.SymbolItem{Code: s.Code, Name: s.Name})
	}
	c.JSON(http.StatusOK, out)
}
