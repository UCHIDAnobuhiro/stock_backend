package twelvedata

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

type logoResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	URL     string `json:"url"`
}

// GetLogoURL はTwelve DataのLogo endpointから株式向けロゴURLを取得します。
func (t *TwelveDataMarket) GetLogoURL(ctx context.Context, symbol string) (string, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	q.Set("apikey", t.cfg.TwelveDataAPIKey)

	u := fmt.Sprintf("%s/logo?%s", t.cfg.BaseURL, q.Encode())
	res, err := t.doRequestWithRetry(ctx, http.MethodGet, u)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			slog.Warn("failed to close response body", "error", err)
		}
	}()

	var body logoResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.Status == "error" {
		return "", fmt.Errorf("twelvedata: %s", body.Message)
	}
	logoURL := strings.TrimSpace(body.URL)
	if logoURL == "" {
		return "", fmt.Errorf("twelvedata: empty logo url for %q", symbol)
	}
	return logoURL, nil
}
