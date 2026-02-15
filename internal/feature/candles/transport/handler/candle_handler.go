// Package handler はcandlesフィーチャーのHTTPハンドラーを提供します。
package handler

import (
	"context"
	"net/http"
	"stock_backend/internal/api"
	"stock_backend/internal/feature/candles/domain/entity"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CandlesUsecase はローソク足データ操作のユースケースインターフェースを定義します。
// Goの慣例に従い、インターフェースは利用者（handler）側で定義します。
type CandlesUsecase interface {
	GetCandles(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
}

// CandlesHandler はローソク足データのHTTPリクエストを処理します。
type CandlesHandler struct {
	uc CandlesUsecase
}

// NewCandlesHandler は指定されたusecaseでCandlesHandlerの新しいインスタンスを生成します。
func NewCandlesHandler(uc CandlesUsecase) *CandlesHandler {
	return &CandlesHandler{uc: uc}
}

// GetCandlesHandler は銘柄コードと時間間隔を受け取り、ローソク足データをJSONで返します。
//
// エンドポイント例:
// GET /candles/:code?interval=1day&outputsize=200
func (h *CandlesHandler) GetCandlesHandler(c *gin.Context) {
	code := c.Param("code")
	// 未指定の場合はデフォルト値を使用
	interval := c.DefaultQuery("interval", "1day")
	outputsizeStr := c.DefaultQuery("outputsize", "200")
	// 文字列を整数に変換
	outputsize, _ := strconv.Atoi(outputsizeStr)

	candles, err := h.uc.GetCandles(c.Request.Context(), code, interval, outputsize)

	if err != nil {
		c.JSON(http.StatusBadGateway, api.ErrorResponse{Error: err.Error()})
		return
	}

	// データをフォーマット
	out := make([]api.CandleResponse, 0, len(candles))
	for _, x := range candles {
		out = append(out, api.CandleResponse{
			Time:   x.Time.UTC().Format("2006-01-02"),
			Open:   x.Open,
			High:   x.High,
			Low:    x.Low,
			Close:  x.Close,
			Volume: x.Volume,
		})
	}

	c.JSON(http.StatusOK, out)
}
