package handler

import (
	"context"
	"net/http"
	"stock_backend/internal/feature/symbollist/domain/entity"
	"stock_backend/internal/feature/symbollist/transport/http/dto"

	"github.com/gin-gonic/gin"
)

// SymbolUsecase は銘柄情報に関するユースケースのインターフェースです。
// Following Go convention: interfaces are defined by the consumer (handler), not the provider (usecase).
type SymbolUsecase interface {
	ListActiveSymbols(ctx context.Context) ([]entity.Symbol, error)
}

// SymbolHandler は銘柄情報に関するHTTPリクエストを処理します。
type SymbolHandler struct {
	uc SymbolUsecase
}

// NewSymbolHandler は新しい SymbolHandler を作成します。
func NewSymbolHandler(uc SymbolUsecase) *SymbolHandler {
	return &SymbolHandler{uc: uc}
}

// List は有効な銘柄の一覧を取得するAPIです。
// Usecaseを呼び出して銘柄一覧を取得し、DTOに変換してJSONレスポンスとして返します。
// Usecaseでエラーが発生した場合は500 Internal Server Errorを返します。
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
