package adapters

import (
	"log/slog"

	"gorm.io/gorm"

	watchlistentity "stock_backend/internal/feature/watchlist/domain/entity"
)

// AddFKConstraints はwatchlistsテーブルのFK制約を冪等に追加します。
// GORMのAutoMigrateはFK制約を自動生成しないため、マイグレーション後に明示的に実行します。
func AddFKConstraints(db *gorm.DB) error {
	if !db.Migrator().HasConstraint(&watchlistentity.UserSymbol{}, "fk_watchlists_user") {
		if err := db.Exec(`ALTER TABLE watchlists ADD CONSTRAINT fk_watchlists_user
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`).Error; err != nil {
			return err
		}
		slog.Info("added FK constraint: fk_watchlists_user")
	}
	if !db.Migrator().HasConstraint(&watchlistentity.UserSymbol{}, "fk_watchlists_symbol") {
		if err := db.Exec(`ALTER TABLE watchlists ADD CONSTRAINT fk_watchlists_symbol
			FOREIGN KEY (symbol_code) REFERENCES symbols(code) ON DELETE RESTRICT`).Error; err != nil {
			return err
		}
		slog.Info("added FK constraint: fk_watchlists_symbol")
	}
	// symbols への FK 参照チェック（ON DELETE RESTRICT）をフルスキャンにしないため、
	// 複合 UNIQUE 先頭ではない symbol_code 単独のインデックスを追加する。
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_watchlists_symbol_code
		ON watchlists (symbol_code)`).Error; err != nil {
		return err
	}
	return nil
}
