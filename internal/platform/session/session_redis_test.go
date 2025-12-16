package session

import (
	"context"
	"stock_backend/internal/feature/auth/domain/entity"
	"stock_backend/internal/feature/auth/usecase"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis creates a miniredis instance for testing.
func setupTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err, "failed to start miniredis")

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	t.Cleanup(func() {
		_ = client.Close()
		mr.Close()
	})

	return client, mr
}

// createTestSession creates a session entity for testing.
func createTestSession(id string, userID uint, expiresIn time.Duration) *entity.Session {
	now := time.Now()
	return &entity.Session{
		ID:        id,
		UserID:    userID,
		UserAgent: "test-agent",
		IPAddress: "127.0.0.1",
		CreatedAt: now,
		ExpiresAt: now.Add(expiresIn),
	}
}

func TestNewSessionRedis(t *testing.T) {
	client, _ := setupTestRedis(t)
	repo := NewSessionRedis(client, "session")

	assert.NotNil(t, repo, "repository is nil")
	assert.NotNil(t, repo.client, "client is nil")
	assert.Equal(t, "session", repo.prefix)
}

func TestSessionRedis_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		session *entity.Session
		wantErr bool
	}{
		{
			name:    "success: create session",
			session: createTestSession("session-001", 1, 7*24*time.Hour),
			wantErr: false,
		},
		{
			name:    "failure: expired session",
			session: createTestSession("expired-session", 1, -1*time.Hour),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, _ := setupTestRedis(t)
			repo := NewSessionRedis(client, "session")

			err := repo.Create(context.Background(), tt.session)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify session exists in Redis
				data, err := client.Get(context.Background(), repo.sessionKey(tt.session.ID)).Result()
				assert.NoError(t, err)
				assert.NotEmpty(t, data)

				// Verify session ID is in user's session set
				isMember, err := client.SIsMember(context.Background(), repo.userSessionsKey(tt.session.UserID), tt.session.ID).Result()
				assert.NoError(t, err)
				assert.True(t, isMember)
			}
		})
	}
}

func TestSessionRedis_FindByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		sessionID   string
		setupFunc   func(t *testing.T, repo *SessionRedis)
		wantErr     bool
		expectedErr error
	}{
		{
			name:      "success: find session",
			sessionID: "find-session-id",
			setupFunc: func(t *testing.T, repo *SessionRedis) {
				session := createTestSession("find-session-id", 1, 7*24*time.Hour)
				err := repo.Create(context.Background(), session)
				require.NoError(t, err)
			},
			wantErr: false,
		},
		{
			name:        "failure: session not found",
			sessionID:   "nonexistent-id",
			wantErr:     true,
			expectedErr: usecase.ErrSessionNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, _ := setupTestRedis(t)
			repo := NewSessionRedis(client, "session")

			if tt.setupFunc != nil {
				tt.setupFunc(t, repo)
			}

			found, err := repo.FindByID(context.Background(), tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, found)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, found)
				assert.Equal(t, tt.sessionID, found.ID)
			}
		})
	}
}

func TestSessionRedis_FindByUserID(t *testing.T) {
	t.Parallel()

	client, _ := setupTestRedis(t)
	repo := NewSessionRedis(client, "session")

	// Create sessions for user 1
	session1 := createTestSession("session-1", 1, 7*24*time.Hour)
	session2 := createTestSession("session-2", 1, 7*24*time.Hour)
	// Create session for user 2
	session3 := createTestSession("session-3", 2, 7*24*time.Hour)

	require.NoError(t, repo.Create(context.Background(), session1))
	require.NoError(t, repo.Create(context.Background(), session2))
	require.NoError(t, repo.Create(context.Background(), session3))

	// Find user 1's sessions
	sessions, err := repo.FindByUserID(context.Background(), 1)
	assert.NoError(t, err)
	assert.Len(t, sessions, 2)

	// Find user 2's sessions
	sessions, err = repo.FindByUserID(context.Background(), 2)
	assert.NoError(t, err)
	assert.Len(t, sessions, 1)

	// Find nonexistent user's sessions
	sessions, err = repo.FindByUserID(context.Background(), 999)
	assert.NoError(t, err)
	assert.Len(t, sessions, 0)
}

func TestSessionRedis_Revoke(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		sessionID   string
		setupFunc   func(t *testing.T, repo *SessionRedis)
		wantErr     bool
		expectedErr error
	}{
		{
			name:      "success: revoke session",
			sessionID: "revoke-session-id",
			setupFunc: func(t *testing.T, repo *SessionRedis) {
				session := createTestSession("revoke-session-id", 1, 7*24*time.Hour)
				err := repo.Create(context.Background(), session)
				require.NoError(t, err)
			},
			wantErr: false,
		},
		{
			name:        "failure: session not found",
			sessionID:   "nonexistent-id",
			wantErr:     true,
			expectedErr: usecase.ErrSessionNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, _ := setupTestRedis(t)
			repo := NewSessionRedis(client, "session")

			if tt.setupFunc != nil {
				tt.setupFunc(t, repo)
			}

			err := repo.Revoke(context.Background(), tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				assert.NoError(t, err)

				// Verify session is revoked
				found, err := repo.FindByID(context.Background(), tt.sessionID)
				assert.NoError(t, err)
				assert.NotNil(t, found.RevokedAt)
			}
		})
	}
}

func TestSessionRedis_RevokeAllByUserID(t *testing.T) {
	t.Parallel()

	client, _ := setupTestRedis(t)
	repo := NewSessionRedis(client, "session")

	// Create sessions for user 1
	session1 := createTestSession("session-1", 1, 7*24*time.Hour)
	session2 := createTestSession("session-2", 1, 7*24*time.Hour)
	// Create session for user 2
	session3 := createTestSession("session-3", 2, 7*24*time.Hour)

	require.NoError(t, repo.Create(context.Background(), session1))
	require.NoError(t, repo.Create(context.Background(), session2))
	require.NoError(t, repo.Create(context.Background(), session3))

	// Revoke all user 1's sessions
	err := repo.RevokeAllByUserID(context.Background(), 1)
	assert.NoError(t, err)

	// Verify user 1's sessions are revoked
	found1, _ := repo.FindByID(context.Background(), "session-1")
	found2, _ := repo.FindByID(context.Background(), "session-2")
	assert.NotNil(t, found1.RevokedAt)
	assert.NotNil(t, found2.RevokedAt)

	// Verify user 2's session is not revoked
	found3, _ := repo.FindByID(context.Background(), "session-3")
	assert.Nil(t, found3.RevokedAt)
}

func TestSessionRedis_CountByUserID(t *testing.T) {
	t.Parallel()

	client, _ := setupTestRedis(t)
	repo := NewSessionRedis(client, "session")

	// Create active sessions
	session1 := createTestSession("active-1", 1, 7*24*time.Hour)
	session2 := createTestSession("active-2", 1, 7*24*time.Hour)

	require.NoError(t, repo.Create(context.Background(), session1))
	require.NoError(t, repo.Create(context.Background(), session2))

	// Revoke one session
	require.NoError(t, repo.Revoke(context.Background(), "active-1"))

	count, err := repo.CountByUserID(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count, "should only count active (non-revoked) sessions")
}

func TestSessionRedis_DeleteOldestByUserID(t *testing.T) {
	t.Parallel()

	client, _ := setupTestRedis(t)
	repo := NewSessionRedis(client, "session")

	// Create sessions with different creation times
	now := time.Now()
	oldSession := &entity.Session{
		ID:        "oldest-session",
		UserID:    1,
		UserAgent: "test",
		IPAddress: "127.0.0.1",
		CreatedAt: now.Add(-2 * time.Hour),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	newSession := &entity.Session{
		ID:        "newest-session",
		UserID:    1,
		UserAgent: "test",
		IPAddress: "127.0.0.1",
		CreatedAt: now.Add(-1 * time.Hour),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}

	require.NoError(t, repo.Create(context.Background(), oldSession))
	require.NoError(t, repo.Create(context.Background(), newSession))

	err := repo.DeleteOldestByUserID(context.Background(), 1)
	assert.NoError(t, err)

	// Verify oldest session is deleted
	_, err = repo.FindByID(context.Background(), "oldest-session")
	assert.ErrorIs(t, err, usecase.ErrSessionNotFound)

	// Verify newest session still exists
	found, err := repo.FindByID(context.Background(), "newest-session")
	assert.NoError(t, err)
	assert.NotNil(t, found)
}

func TestSessionRedis_DeleteExpired(t *testing.T) {
	t.Parallel()

	client, _ := setupTestRedis(t)
	repo := NewSessionRedis(client, "session")

	// DeleteExpired is a no-op for Redis (TTL handles it)
	deleted, err := repo.DeleteExpired(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
}

func TestSessionRedis_KeyGeneration(t *testing.T) {
	t.Parallel()

	client, _ := setupTestRedis(t)
	repo := NewSessionRedis(client, "test-prefix")

	assert.Equal(t, "test-prefix:session-id", repo.sessionKey("session-id"))
	assert.Equal(t, "test-prefix:user:123", repo.userSessionsKey(123))
}
