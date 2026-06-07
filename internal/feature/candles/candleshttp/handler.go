package candleshttp

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/UCHIDAnobuhiro/stock-backend/internal/api"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/candles"
)

// Usecase はローソク足データ操作のユースケースインターフェースを定義します。
// Goの慣例に従い、インターフェースは利用者（handler）側で定義します。
type Usecase interface {
	GetCandles(ctx context.Context, symbol, interval string, outputsize int) ([]candles.Candle, error)
}

// Handler はローソク足データのHTTPリクエストを処理します。
type Handler struct {
	uc Usecase
}

// NewHandler は指定されたusecaseでHandlerの新しいインスタンスを生成します。
func NewHandler(uc Usecase) *Handler {
	return &Handler{uc: uc}
}

// GetCandlesHandler は銘柄コードと時間間隔を受け取り、ローソク足データをJSONで返します。
//
// エンドポイント例:
// GET /candles/:code?interval=1day&outputsize=200
func (h *Handler) GetCandlesHandler(c *gin.Context) {
	code := c.Param("code")
	// 未指定の場合はデフォルト値を使用
	interval := c.DefaultQuery("interval", "1day")
	outputsizeStr := c.DefaultQuery("outputsize", "200")
	// 文字列を整数に変換
	outputsize, err := strconv.Atoi(outputsizeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "outputsize must be an integer"})
		return
	}

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
