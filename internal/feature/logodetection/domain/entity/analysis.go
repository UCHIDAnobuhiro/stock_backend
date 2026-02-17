package entity

// CompanyAnalysis は企業の分析結果を表します。
type CompanyAnalysis struct {
	CompanyName string // 分析対象の企業名
	Summary     string // AI生成の分析サマリー
}
