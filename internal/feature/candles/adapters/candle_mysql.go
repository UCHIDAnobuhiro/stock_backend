// Package adapters はcandlesフィーチャーのリポジトリ実装を提供します。
package adapters

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"stock_backend/internal/feature/candles/domain/entity"
	"stock_backend/internal/feature/candles/usecase"
)

// candleMySQL はCandleRepositoryインターフェースのMySQL実装です。
type candleMySQL struct {
	db *gorm.DB
}

// candleMySQLがCandleRepositoryを実装していることをコンパイル時に検証します。
var _ usecase.CandleRepository = (*candleMySQL)(nil)

// NewCandleRepository は指定されたデータベース接続でcandleMySQLリポジトリの新しいインスタンスを生成します。
func NewCandleRepository(db *gorm.DB) *candleMySQL {
	return &candleMySQL{db: db}
}

// CandleModel はローソク足データのデータベースモデルです。
// Upsert操作のために（symbol, interval, time）の複合ユニークインデックスを使用します。
type CandleModel struct {
	ID       uint      `gorm:"primaryKey"`
	Symbol   string    `gorm:"size:32;not null;uniqueIndex:candle_sym_int_time,priority:1"`
	Interval string    `gorm:"size:16;not null;uniqueIndex:candle_sym_int_time,priority:2"`
	Time     time.Time `gorm:"not null;uniqueIndex:candle_sym_int_time,priority:3"`

	Open   float64 `gorm:"not null"`
	High   float64 `gorm:"not null"`
	Low    float64 `gorm:"not null"`
	Close  float64 `gorm:"not null"`
	Volume int64   `gorm:"not null;default:0"`
}

// TableName はこのモデルのデータベーステーブル名を返します。
func (CandleModel) TableName() string {
	return "candles"
}

// toModel はドメインエンティティをデータベースモデルに変換します。
func toModel(e entity.Candle) CandleModel {
	return CandleModel{
		Symbol:   e.Symbol,
		Interval: e.Interval,
		Time:     e.Time,
		Open:     e.Open,
		High:     e.High,
		Low:      e.Low,
		Close:    e.Close,
		Volume:   e.Volume,
	}
}

// UpsertBatch はローソク足データをバッチで挿入または更新します。
// MySQLのON DUPLICATE KEY UPDATEを使用して効率的なUpsertを行います。
func (r *candleMySQL) UpsertBatch(ctx context.Context, candles []entity.Candle) error {
	if len(candles) == 0 {
		return nil
	}
	ms := make([]CandleModel, 0, len(candles))
	for _, e := range candles {
		ms = append(ms, toModel(e))
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "symbol"}, {Name: "interval"}, {Name: "time"}},
		DoUpdates: clause.AssignmentColumns([]string{"open", "high", "low", "close", "volume"}),
	}).Create(&ms).Error
}

// Find は指定された銘柄とインターバルのローソク足データを取得します。
// 結果は時間の降順でソートされ、outputsizeで件数が制限されます。
func (r *candleMySQL) Find(ctx context.Context, symbol, interval string, outputsize int) ([]entity.Candle, error) {
	var rows []CandleModel
	q := r.db.WithContext(ctx).
		Where("symbol = ? AND `interval` = ?", symbol, interval).
		Order("`time` DESC")
	if outputsize > 0 {
		q = q.Limit(outputsize)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]entity.Candle, 0, len(rows))
	for _, m := range rows {
		out = append(out, entity.Candle{
			Symbol:   m.Symbol,
			Interval: m.Interval,
			Time:     m.Time,
			Open:     m.Open,
			High:     m.High,
			Low:      m.Low,
			Close:    m.Close,
			Volume:   m.Volume,
		})
	}
	return out, nil
}
