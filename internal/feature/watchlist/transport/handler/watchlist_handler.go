// Package handler はwatchlistフィーチャーのHTTPハンドラーを提供します。
package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"stock_backend/internal/api"
	"stock_backend/internal/feature/watchlist/domain/entity"
	"stock_backend/internal/feature/watchlist/usecase"
	jwtmw "stock_backend/internal/platform/jwt"
)

// WatchlistUsecase はウォッチリスト操作のユースケースインターフェースを定義します。
// Goの慣例に従い、インターフェースは利用者（handler）側で定義します。
type WatchlistUsecase interface {
	ListUserSymbols(ctx context.Context, userID uint) ([]entity.UserSymbol, error)
	AddSymbol(ctx context.Context, userID uint, symbolCode string) error
	RemoveSymbol(ctx context.Context, userID uint, symbolCode string) error
	ReorderSymbols(ctx context.Context, userID uint, codeOrder []string) error
}

// WatchlistHandler はウォッチリストに関連するHTTPリクエストを処理します。
type WatchlistHandler struct {
	uc WatchlistUsecase
}

// NewWatchlistHandler はWatchlistHandlerの新しいインスタンスを生成します。
func NewWatchlistHandler(uc WatchlistUsecase) *WatchlistHandler {
	return &WatchlistHandler{uc: uc}
}

// getUserID はGinコンテキストからuserIDを安全に取得します。
// 取得できない場合はfalseを返し、呼び出し元で401を返すべきです。
func getUserID(c *gin.Context) (uint, bool) {
	v, ok := c.Get(jwtmw.ContextUserID)
	if !ok {
		return 0, false
	}
	userID, ok := v.(uint)
	return userID, ok
}

// List はユーザーのウォッチリスト銘柄一覧を取得します。
func (h *WatchlistHandler) List(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: "unauthorized"})
		return
	}

	symbols, err := h.uc.ListUserSymbols(c.Request.Context(), userID)
	if err != nil {
		slog.Error("failed to list watchlist", "error", err, "userID", userID)
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: "failed to get watchlist"})
		return
	}

	out := make([]api.WatchlistItem, 0, len(symbols))
	for _, s := range symbols {
		out = append(out, api.WatchlistItem{
			SymbolCode: s.SymbolCode,
			SortKey:    s.SortKey,
		})
	}
	c.JSON(http.StatusOK, out)
}

// Add はユーザーのウォッチリストに銘柄を追加します。
func (h *WatchlistHandler) Add(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: "unauthorized"})
		return
	}

	var req api.AddWatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "invalid request"})
		return
	}

	if err := h.uc.AddSymbol(c.Request.Context(), userID, req.SymbolCode); err != nil {
		if errors.Is(err, usecase.ErrSymbolAlreadyExists) {
			c.JSON(http.StatusConflict, api.ErrorResponse{Error: "symbol already exists in watchlist"})
			return
		}
		if errors.Is(err, usecase.ErrSymbolNotInMaster) {
			c.JSON(http.StatusNotFound, api.ErrorResponse{Error: "symbol not found in master symbol list"})
			return
		}
		slog.Error("failed to add watchlist symbol", "error", err, "userID", userID)
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: "failed to add symbol"})
		return
	}

	c.JSON(http.StatusCreated, api.MessageResponse{Message: "ok"})
}

// Remove はユーザーのウォッチリストから銘柄を削除します。
func (h *WatchlistHandler) Remove(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: "unauthorized"})
		return
	}
	code := c.Param("code")

	if err := h.uc.RemoveSymbol(c.Request.Context(), userID, code); err != nil {
		if errors.Is(err, usecase.ErrSymbolNotFound) {
			c.JSON(http.StatusNotFound, api.ErrorResponse{Error: "symbol not found in watchlist"})
			return
		}
		slog.Error("failed to remove watchlist symbol", "error", err, "userID", userID)
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: "failed to remove symbol"})
		return
	}

	c.Status(http.StatusNoContent)
}

// Reorder はユーザーのウォッチリスト銘柄の並び順を更新します。
func (h *WatchlistHandler) Reorder(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: "unauthorized"})
		return
	}

	var req api.ReorderWatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "invalid request"})
		return
	}

	if err := h.uc.ReorderSymbols(c.Request.Context(), userID, req.SymbolCodes); err != nil {
		if errors.Is(err, usecase.ErrInvalidReorder) {
			c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: err.Error()})
			return
		}
		slog.Error("failed to reorder watchlist", "error", err, "userID", userID)
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: "failed to reorder"})
		return
	}

	c.JSON(http.StatusOK, api.MessageResponse{Message: "ok"})
}
