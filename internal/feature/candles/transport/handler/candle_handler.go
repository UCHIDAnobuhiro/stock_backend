// internal/handler/candles.go
package handler

import (
	"context"
	"net/http"
	"stock_backend/internal/feature/candles/domain/entity"
	"stock_backend/internal/feature/candles/transport/http/dto"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CandlesUsecase defines the use case interface for candlestick data operations.
// Following Go convention: interfaces are defined by the consumer (handler), not the provider (usecase).
type CandlesUsecase interface {
	GetCandles(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error)
}

type CandlesHandler struct {
	uc CandlesUsecase
}

func NewCandlesHandler(uc CandlesUsecase) *CandlesHandler {
	return &CandlesHandler{uc: uc}
}

// GetCandlesHandler receives a stock symbol and time interval, then returns candlestick data as JSON.
//
// Endpoint example:
// GET /candles/:code?interval=1day&outputsize=200
func (h *CandlesHandler) GetCandlesHandler(c *gin.Context) {
	code := c.Param("code")
	// Use defaults if not specified
	interval := c.DefaultQuery("interval", "1day")
	outputsizeStr := c.DefaultQuery("outputsize", "200")
	// Convert string to integer
	outputsize, _ := strconv.Atoi(outputsizeStr)

	candles, err := h.uc.GetCandles(c.Request.Context(), code, interval, outputsize)

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// Format data
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
