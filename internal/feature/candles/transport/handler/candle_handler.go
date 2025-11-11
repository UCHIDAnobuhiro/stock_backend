// internal/handler/candles.go
package handler

import (
	"net/http"
	"stock_backend/internal/feature/candles/transport/http/dto"
	"stock_backend/internal/feature/candles/usecase"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CandlesHandler struct {
	uc usecase.CandlesUsecase
}

func NewCandlesHandler(uc usecase.CandlesUsecase) *CandlesHandler {
	return &CandlesHandler{uc: uc}
}

// GetCandles は銘柄コードと時間足を受け取り、ロウソク足データを JSON で返します。
//
// エンドポイント例
// GET /candles/:code?interval=1day&outputsize=200
func (h *CandlesHandler) GetCandlesHandler(c *gin.Context) {
	code := c.Param("code")
	// 指定されていない場合は以下を設定
	interval := c.DefaultQuery("interval", "1day")
	outputsizeStr := c.DefaultQuery("outputsize", "200")
	// 文字列を数値に変換
	outputsize, _ := strconv.Atoi(outputsizeStr)

	candles, err := h.uc.GetCandles(c.Request.Context(), code, interval, outputsize)

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// データの整形
	out := make([]dto.CandleResponse, 0, len(candles))
	for _, x := range candles {
		out = append(out, dto.CandleResponse{
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
