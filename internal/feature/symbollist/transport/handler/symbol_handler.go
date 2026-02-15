// Package handler はsymbollistフィーチャーのHTTPハンドラーを提供します。
package handler

import (
	"context"
	"net/http"
	"stock_backend/internal/feature/symbollist/domain/entity"
	"stock_backend/internal/api"

	"github.com/gin-gonic/gin"
)

// SymbolUsecase は銘柄（株式コード）操作のユースケースインターフェースを定義します。
// Goの慣例に従い、インターフェースは利用者（handler）側で定義します。
type SymbolUsecase interface {
	ListActiveSymbols(ctx context.Context) ([]entity.Symbol, error)
}

// SymbolHandler は銘柄情報に関連するHTTPリクエストを処理します。
type SymbolHandler struct {
	uc SymbolUsecase
}

// NewSymbolHandler はSymbolHandlerの新しいインスタンスを生成します。
func NewSymbolHandler(uc SymbolUsecase) *SymbolHandler {
	return &SymbolHandler{uc: uc}
}

// List はアクティブな銘柄の一覧を取得します。
// ユースケースを呼び出して銘柄リストを取得し、DTOに変換してJSONレスポンスとして返します。
// ユースケースがエラーを返した場合は500 Internal Server Errorを返します。
func (h *SymbolHandler) List(c *gin.Context) {
	symbols, err := h.uc.ListActiveSymbols(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
		return
	}
	out := make([]api.SymbolItem, 0, len(symbols))
	for _, s := range symbols {
		out = append(out, api.SymbolItem{Code: s.Code, Name: s.Name})
	}
	c.JSON(http.StatusOK, out)
}
