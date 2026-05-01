// Package adapters はauthフィーチャーのリポジトリ実装を提供します。
package adapters

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"stock_backend/internal/feature/auth/usecase"
)

// redisOAuthStateStore はOAuthStateStoreインターフェースのRedis実装です。
type redisOAuthStateStore struct {
	rdb *redis.Client
}

var _ usecase.OAuthStateStore = (*redisOAuthStateStore)(nil)

// NewRedisOAuthStateStore は指定されたRedisクライアントでredisOAuthStateStoreを生成します。
func NewRedisOAuthStateStore(rdb *redis.Client) *redisOAuthStateStore {
	return &redisOAuthStateStore{rdb: rdb}
}

func stateKey(state string) string {
	return fmt.Sprintf("oauth:state:%s", state)
}

// SaveState はstateとcodeVerifierをTTL付きでRedisに保存します。
func (s *redisOAuthStateStore) SaveState(ctx context.Context, state, codeVerifier string, ttl time.Duration) error {
	return s.rdb.Set(ctx, stateKey(state), codeVerifier, ttl).Err()
}

// ConsumeState はstateに対応するcodeVerifierを取得して削除します（GETDEL: atomic）。
// stateが存在しない・期限切れの場合はErrStateNotFoundを返します。
func (s *redisOAuthStateStore) ConsumeState(ctx context.Context, state string) (string, error) {
	val, err := s.rdb.GetDel(ctx, stateKey(state)).Result()
	if errors.Is(err, redis.Nil) {
		return "", usecase.ErrStateNotFound
	}
	if err != nil {
		return "", fmt.Errorf("state store error: %w", err)
	}
	return val, nil
}
