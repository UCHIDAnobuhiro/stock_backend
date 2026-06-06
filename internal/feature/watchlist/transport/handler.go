package watchlisthttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"stock_backend/internal/api"
	"stock_backend/internal/feature/watchlist"
	jwtmw "stock_backend/internal/platform/jwt"
)

// WatchlistUsecase はウォッチリスト操作のユースケースインターフェースを定義します。
type WatchlistUsecase interface {
	ListUserSymbols(ctx context.Context, userID int64) ([]watchlist.UserSymbol, error)
	AddSymbol(ctx context.Context, userID int64, symbolCode string) error
	RemoveSymbol(ctx context.Context, userID int64, symbolCode string) error
	ReorderSymbols(ctx context.Context, userID int64, orderedCodes []string) error
}

// WatchlistHandler はウォッチリストに関連するHTTPリクエストを処理します。
type WatchlistHandler struct {
	uc WatchlistUsecase
}

// NewWatchlistHandler はWatchlistHandlerの新しいインスタンスを生成します。
func NewWatchlistHandler(uc WatchlistUsecase) *WatchlistHandler {
	return &WatchlistHandler{uc: uc}
}

// List はユーザーのウォッチリスト一覧を取得します。
func (h *WatchlistHandler) List(c *gin.Context) {
	userID := c.MustGet(jwtmw.ContextUserID).(int64)

	entries, err := h.uc.ListUserSymbols(c.Request.Context(), userID)
	if err != nil {
		slog.Error("failed to list watchlist", "error", err, "userID", userID)
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: "internal server error"})
		return
	}

	out := make([]api.WatchlistItem, 0, len(entries))
	for _, e := range entries {
		out = append(out, api.WatchlistItem{
			Id:         e.ID,
			SymbolCode: e.SymbolCode,
			SortKey:    e.SortKey,
		})
	}
	c.JSON(http.StatusOK, out)
}

// Add はウォッチリストに銘柄を追加します。
func (h *WatchlistHandler) Add(c *gin.Context) {
	userID := c.MustGet(jwtmw.ContextUserID).(int64)

	var req api.AddWatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.uc.AddSymbol(c.Request.Context(), userID, req.SymbolCode); err != nil {
		switch {
		case errors.Is(err, watchlist.ErrSymbolNotFound):
			c.JSON(http.StatusNotFound, api.ErrorResponse{Error: err.Error()})
		case errors.Is(err, watchlist.ErrAlreadyInWatchlist):
			c.JSON(http.StatusConflict, api.ErrorResponse{Error: err.Error()})
		default:
			slog.Error("failed to add watchlist symbol", "error", err, "userID", userID)
			c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: "internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, api.MessageResponse{Message: "added to watchlist"})
}

// Remove はウォッチリストから銘柄を削除します。
func (h *WatchlistHandler) Remove(c *gin.Context) {
	userID := c.MustGet(jwtmw.ContextUserID).(int64)
	code := c.Param("code")

	if err := h.uc.RemoveSymbol(c.Request.Context(), userID, code); err != nil {
		switch {
		case errors.Is(err, watchlist.ErrNotInWatchlist):
			c.JSON(http.StatusNotFound, api.ErrorResponse{Error: err.Error()})
		default:
			slog.Error("failed to remove watchlist symbol", "error", err, "userID", userID)
			c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: "internal server error"})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// Reorder はウォッチリストの並び順を更新します。
func (h *WatchlistHandler) Reorder(c *gin.Context) {
	userID := c.MustGet(jwtmw.ContextUserID).(int64)

	var req api.ReorderWatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.uc.ReorderSymbols(c.Request.Context(), userID, req.Codes); err != nil {
		slog.Error("failed to reorder watchlist", "error", err, "userID", userID)
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: "internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}
