package symbollisthttp

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"stock_backend/internal/api"
	"stock_backend/internal/feature/symbollist"
)

// Usecase は銘柄（株式コード）操作のユースケースインターフェースを定義します。
// Goの慣例に従い、インターフェースは利用者（handler）側で定義します。
type Usecase interface {
	ListActiveSymbols(ctx context.Context) ([]symbollist.Symbol, error)
}

// Handler は銘柄情報に関連するHTTPリクエストを処理します。
type Handler struct {
	uc Usecase
}

// NewHandler はHandlerの新しいインスタンスを生成します。
func NewHandler(uc Usecase) *Handler {
	return &Handler{uc: uc}
}

// List はアクティブな銘柄の一覧を取得します。
// ユースケースを呼び出して銘柄リストを取得し、DTOに変換してJSONレスポンスとして返します。
// ユースケースがエラーを返した場合は500 Internal Server Errorを返します。
func (h *Handler) List(c *gin.Context) {
	symbols, err := h.uc.ListActiveSymbols(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
		return
	}
	out := make([]api.SymbolItem, 0, len(symbols))
	for _, s := range symbols {
		out = append(out, api.SymbolItem{Code: s.Code, Name: s.Name, LogoUrl: s.LogoURL})
	}
	c.JSON(http.StatusOK, out)
}
