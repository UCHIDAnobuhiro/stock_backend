package mysql

import (
	"context"
	"stock_backend/internal/domain/entity"
	"stock_backend/internal/domain/repository"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type candleMySQL struct {
	db *gorm.DB
}

var _ repository.CandleRepository = (*candleMySQL)(nil)

func NewCandleRepository(db *gorm.DB) repository.CandleRepository {
	return &candleMySQL{db: db}
}

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

func (CandleModel) TableName() string {
	return "candles"
}

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
