package handler

import (
	"net/http"
	"stock_backend/internal/interface/dto"
	"stock_backend/internal/usecase"

	"github.com/gin-gonic/gin"
)

type SymbolHandler struct {
	uc *usecase.SymbolUsecase
}

func NewSymbolHandler(uc *usecase.SymbolUsecase) *SymbolHandler {
	return &SymbolHandler{uc: uc}
}

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
