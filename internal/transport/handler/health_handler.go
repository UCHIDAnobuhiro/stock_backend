// Package handler はプラットフォームレベルのエンドポイント用HTTPハンドラーを提供します。
package handler

import (
	"net/http"

	"github.com/UCHIDAnobuhiro/stock-backend/internal/api"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/transport/httpx"
)

// Health はサービスヘルスチェック用の /healthz エンドポイントを処理します。
// HTTPメソッドに応じて適切にレスポンスし、キャッシュを防止します。
func Health(w http.ResponseWriter, r *http.Request) {
	// 明示的にキャッシュを防止
	w.Header().Set("Cache-Control", "no-store")

	// すべてのGET/HEAD/OPTIONSリクエストに対して200または204を返す
	switch r.Method {
	case http.MethodHead:
		w.WriteHeader(http.StatusOK)
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
	default:
		httpx.WriteJSON(w, http.StatusOK, api.HealthResponse{Status: "ok"})
	}
}
