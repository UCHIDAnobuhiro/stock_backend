package adapters

import (
	"log/slog"

	"gorm.io/gorm"
)

// AddFKConstraints は candles テーブルの FK 制約を冪等に追加します。
// GORMのAutoMigrateはFK制約を自動生成しないため、マイグレーション後に明示的に実行します。
func AddFKConstraints(db *gorm.DB) error {
	if !db.Migrator().HasConstraint(&CandleModel{}, "fk_candles_symbol") {
		if err := db.Exec(`ALTER TABLE candles ADD CONSTRAINT fk_candles_symbol
			FOREIGN KEY (symbol_code) REFERENCES symbols(code) ON DELETE RESTRICT`).Error; err != nil {
			return err
		}
		slog.Info("added FK constraint: fk_candles_symbol")
	}
	return nil
}
