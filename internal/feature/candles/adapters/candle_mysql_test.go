package adapters

import (
	"context"
	"stock_backend/internal/feature/candles/domain/entity"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB prepares an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to initialize test database")

	err = db.AutoMigrate(&CandleModel{})
	require.NoError(t, err, "failed to migrate table")

	return db
}

// seedCandle creates a test candle in the database for testing.
func seedCandle(t *testing.T, db *gorm.DB, symbol, interval string, time time.Time) *CandleModel {
	t.Helper()

	candle := &CandleModel{
		Symbol:   symbol,
		Interval: interval,
		Time:     time,
		Open:     100.0,
		High:     110.0,
		Low:      90.0,
		Close:    105.0,
		Volume:   1000,
	}
	err := db.Create(candle).Error
	require.NoError(t, err, "failed to seed candle")

	return candle
}

func TestNewCandleRepository(t *testing.T) {
	db := setupTestDB(t)

	repo := NewCandleRepository(db)

	assert.NotNil(t, repo, "repository is nil")
	assert.NotNil(t, repo.db, "database connection is nil")
}

func TestCandleMySQL_UpsertBatch(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		candles      []entity.Candle
		wantErr      bool
		setupFunc    func(t *testing.T, db *gorm.DB)
		validateFunc func(t *testing.T, db *gorm.DB)
	}{
		{
			name: "success: insert single candle",
			candles: []entity.Candle{
				{
					Symbol:   "AAPL",
					Interval: "1day",
					Time:     baseTime,
					Open:     100.0,
					High:     110.0,
					Low:      90.0,
					Close:    105.0,
					Volume:   1000,
				},
			},
			wantErr: false,
			validateFunc: func(t *testing.T, db *gorm.DB) {
				var count int64
				db.Model(&CandleModel{}).Count(&count)
				assert.Equal(t, int64(1), count, "candle count does not match")
			},
		},
		{
			name: "success: insert multiple candles",
			candles: []entity.Candle{
				{
					Symbol:   "AAPL",
					Interval: "1day",
					Time:     baseTime,
					Open:     100.0,
					High:     110.0,
					Low:      90.0,
					Close:    105.0,
					Volume:   1000,
				},
				{
					Symbol:   "AAPL",
					Interval: "1day",
					Time:     baseTime.AddDate(0, 0, 1),
					Open:     105.0,
					High:     115.0,
					Low:      95.0,
					Close:    110.0,
					Volume:   1500,
				},
			},
			wantErr: false,
			validateFunc: func(t *testing.T, db *gorm.DB) {
				var count int64
				db.Model(&CandleModel{}).Count(&count)
				assert.Equal(t, int64(2), count, "candle count does not match")
			},
		},
		{
			name:    "success: empty slice",
			candles: []entity.Candle{},
			wantErr: false,
			validateFunc: func(t *testing.T, db *gorm.DB) {
				var count int64
				db.Model(&CandleModel{}).Count(&count)
				assert.Equal(t, int64(0), count, "candle count should be 0")
			},
		},
		{
			name: "success: upsert updates existing candle",
			candles: []entity.Candle{
				{
					Symbol:   "AAPL",
					Interval: "1day",
					Time:     baseTime,
					Open:     200.0,
					High:     220.0,
					Low:      180.0,
					Close:    210.0,
					Volume:   2000,
				},
			},
			wantErr: false,
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedCandle(t, db, "AAPL", "1day", baseTime)
			},
			validateFunc: func(t *testing.T, db *gorm.DB) {
				var count int64
				db.Model(&CandleModel{}).Count(&count)
				assert.Equal(t, int64(1), count, "candle count should remain 1 after upsert")

				var candle CandleModel
				db.First(&candle)
				assert.Equal(t, 200.0, candle.Open, "Open should be updated")
				assert.Equal(t, 220.0, candle.High, "High should be updated")
				assert.Equal(t, 180.0, candle.Low, "Low should be updated")
				assert.Equal(t, 210.0, candle.Close, "Close should be updated")
				assert.Equal(t, int64(2000), candle.Volume, "Volume should be updated")
			},
		},
		{
			name: "success: upsert with mixed insert and update",
			candles: []entity.Candle{
				{
					Symbol:   "AAPL",
					Interval: "1day",
					Time:     baseTime,
					Open:     200.0,
					High:     220.0,
					Low:      180.0,
					Close:    210.0,
					Volume:   2000,
				},
				{
					Symbol:   "AAPL",
					Interval: "1day",
					Time:     baseTime.AddDate(0, 0, 1),
					Open:     210.0,
					High:     230.0,
					Low:      190.0,
					Close:    220.0,
					Volume:   2500,
				},
			},
			wantErr: false,
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedCandle(t, db, "AAPL", "1day", baseTime)
			},
			validateFunc: func(t *testing.T, db *gorm.DB) {
				var count int64
				db.Model(&CandleModel{}).Count(&count)
				assert.Equal(t, int64(2), count, "candle count should be 2")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewCandleRepository(db)

			if tt.setupFunc != nil {
				tt.setupFunc(t, db)
			}

			err := repo.UpsertBatch(context.Background(), tt.candles)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validateFunc != nil {
					tt.validateFunc(t, db)
				}
			}
		})
	}
}

func TestCandleMySQL_Find(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		symbol       string
		interval     string
		outputsize   int
		wantErr      bool
		setupFunc    func(t *testing.T, db *gorm.DB)
		validateFunc func(t *testing.T, candles []entity.Candle)
	}{
		{
			name:       "success: find candles by symbol and interval",
			symbol:     "AAPL",
			interval:   "1day",
			outputsize: 10,
			wantErr:    false,
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedCandle(t, db, "AAPL", "1day", baseTime)
				seedCandle(t, db, "AAPL", "1day", baseTime.AddDate(0, 0, 1))
			},
			validateFunc: func(t *testing.T, candles []entity.Candle) {
				assert.Len(t, candles, 2, "should return 2 candles")
			},
		},
		{
			name:       "success: empty result when no matching candles",
			symbol:     "NOTFOUND",
			interval:   "1day",
			outputsize: 10,
			wantErr:    false,
			validateFunc: func(t *testing.T, candles []entity.Candle) {
				assert.Empty(t, candles, "should return empty slice")
			},
		},
		{
			name:       "success: filter by symbol only",
			symbol:     "AAPL",
			interval:   "1day",
			outputsize: 10,
			wantErr:    false,
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedCandle(t, db, "AAPL", "1day", baseTime)
				seedCandle(t, db, "GOOGL", "1day", baseTime)
			},
			validateFunc: func(t *testing.T, candles []entity.Candle) {
				assert.Len(t, candles, 1, "should return only AAPL candle")
				assert.Equal(t, "AAPL", candles[0].Symbol)
			},
		},
		{
			name:       "success: filter by interval",
			symbol:     "AAPL",
			interval:   "1day",
			outputsize: 10,
			wantErr:    false,
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedCandle(t, db, "AAPL", "1day", baseTime)
				seedCandle(t, db, "AAPL", "1week", baseTime)
			},
			validateFunc: func(t *testing.T, candles []entity.Candle) {
				assert.Len(t, candles, 1, "should return only 1day interval")
				assert.Equal(t, "1day", candles[0].Interval)
			},
		},
		{
			name:       "success: respect outputsize limit",
			symbol:     "AAPL",
			interval:   "1day",
			outputsize: 2,
			wantErr:    false,
			setupFunc: func(t *testing.T, db *gorm.DB) {
				for i := 0; i < 5; i++ {
					seedCandle(t, db, "AAPL", "1day", baseTime.AddDate(0, 0, i))
				}
			},
			validateFunc: func(t *testing.T, candles []entity.Candle) {
				assert.Len(t, candles, 2, "should return only 2 candles")
			},
		},
		{
			name:       "success: outputsize 0 returns all",
			symbol:     "AAPL",
			interval:   "1day",
			outputsize: 0,
			wantErr:    false,
			setupFunc: func(t *testing.T, db *gorm.DB) {
				for i := 0; i < 5; i++ {
					seedCandle(t, db, "AAPL", "1day", baseTime.AddDate(0, 0, i))
				}
			},
			validateFunc: func(t *testing.T, candles []entity.Candle) {
				assert.Len(t, candles, 5, "should return all candles")
			},
		},
		{
			name:       "success: results ordered by time descending",
			symbol:     "AAPL",
			interval:   "1day",
			outputsize: 10,
			wantErr:    false,
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedCandle(t, db, "AAPL", "1day", baseTime)
				seedCandle(t, db, "AAPL", "1day", baseTime.AddDate(0, 0, 2))
				seedCandle(t, db, "AAPL", "1day", baseTime.AddDate(0, 0, 1))
			},
			validateFunc: func(t *testing.T, candles []entity.Candle) {
				assert.Len(t, candles, 3, "should return 3 candles")
				assert.True(t, candles[0].Time.After(candles[1].Time), "first should be newer than second")
				assert.True(t, candles[1].Time.After(candles[2].Time), "second should be newer than third")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewCandleRepository(db)

			if tt.setupFunc != nil {
				tt.setupFunc(t, db)
			}

			candles, err := repo.Find(context.Background(), tt.symbol, tt.interval, tt.outputsize)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validateFunc != nil {
					tt.validateFunc(t, candles)
				}
			}
		})
	}
}

func TestCandleMySQL_Find_EntityMapping(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewCandleRepository(db)

	testTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	candle := &CandleModel{
		Symbol:   "AAPL",
		Interval: "1day",
		Time:     testTime,
		Open:     150.5,
		High:     155.75,
		Low:      149.25,
		Close:    154.0,
		Volume:   5000000,
	}
	err := db.Create(candle).Error
	require.NoError(t, err)

	result, err := repo.Find(context.Background(), "AAPL", "1day", 1)
	require.NoError(t, err)
	require.Len(t, result, 1)

	assert.Equal(t, "AAPL", result[0].Symbol, "Symbol does not match")
	assert.Equal(t, "1day", result[0].Interval, "Interval does not match")
	assert.Equal(t, testTime.Unix(), result[0].Time.Unix(), "Time does not match")
	assert.Equal(t, 150.5, result[0].Open, "Open does not match")
	assert.Equal(t, 155.75, result[0].High, "High does not match")
	assert.Equal(t, 149.25, result[0].Low, "Low does not match")
	assert.Equal(t, 154.0, result[0].Close, "Close does not match")
	assert.Equal(t, int64(5000000), result[0].Volume, "Volume does not match")
}
