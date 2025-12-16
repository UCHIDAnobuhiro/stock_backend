package di

import (
	authadapters "stock_backend/internal/feature/auth/adapters"
	"stock_backend/internal/feature/auth/usecase"
	"stock_backend/internal/platform/session"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// NewSessionRepository creates a SessionRepository implementation.
// If Redis is available, it returns a Redis-backed implementation.
// Otherwise, it falls back to MySQL.
func NewSessionRepository(rdb *redis.Client, db *gorm.DB) usecase.SessionRepository {
	if rdb != nil {
		return session.NewSessionRedis(rdb, "session")
	}
	return authadapters.NewSessionMySQL(db)
}
