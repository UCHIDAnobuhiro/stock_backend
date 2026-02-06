// Package dto はsymbollist HTTP APIのデータ転送オブジェクトを定義します。
package dto

// SymbolItem はAPIレスポンスにおける銘柄を表します。
// クライアントに必要な公開フィールドのみを含みます。
type SymbolItem struct {
	Code string `json:"code"` // 銘柄コード（例: "AAPL", "7203.T"）
	Name string `json:"name"` // 企業名
}
