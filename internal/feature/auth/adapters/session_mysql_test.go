package adapters

import (
	"context"
	"stock_backend/internal/feature/auth/domain/entity"
	"stock_backend/internal/feature/auth/usecase"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupSessionTestDB prepares an in-memory SQLite database for session testing.
func setupSessionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to initialize test database")

	// Create Session table
	err = db.AutoMigrate(&SessionModel{})
	require.NoError(t, err, "failed to migrate table")

	return db
}

// seedSession creates a test session in the database for testing.
func seedSession(t *testing.T, db *gorm.DB, id string, userID uint, expiresAt time.Time, revokedAt *time.Time) *entity.Session {
	t.Helper()

	now := time.Now()
	session := &SessionModel{
		ID:        id,
		UserID:    userID,
		UserAgent: "test-agent",
		IPAddress: "127.0.0.1",
		CreatedAt: now,
		ExpiresAt: expiresAt,
		RevokedAt: revokedAt,
	}
	err := db.Create(session).Error
	require.NoError(t, err, "failed to seed session")

	return session.ToEntity()
}

func TestNewSessionMySQL(t *testing.T) {
	db := setupSessionTestDB(t)

	repo := NewSessionMySQL(db)

	assert.NotNil(t, repo, "repository is nil")
	assert.NotNil(t, repo.db, "database connection is nil")
}

func TestSessionMySQL_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		session      *entity.Session
		wantErr      bool
		validateFunc func(t *testing.T, db *gorm.DB, session *entity.Session)
	}{
		{
			name: "success: session creation",
			session: &entity.Session{
				ID:        "test-session-id-001",
				UserID:    1,
				UserAgent: "Mozilla/5.0",
				IPAddress: "192.168.1.1",
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			},
			wantErr: false,
			validateFunc: func(t *testing.T, db *gorm.DB, session *entity.Session) {
				var found SessionModel
				err := db.Where("id = ?", session.ID).First(&found).Error
				assert.NoError(t, err)
				assert.Equal(t, session.UserID, found.UserID)
				assert.Equal(t, session.UserAgent, found.UserAgent)
			},
		},
		{
			name: "failure: duplicate session ID",
			session: &entity.Session{
				ID:        "duplicate-id",
				UserID:    1,
				UserAgent: "test-agent",
				IPAddress: "127.0.0.1",
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupSessionTestDB(t)
			repo := NewSessionMySQL(db)

			// Setup duplicate for duplicate test
			if tt.name == "failure: duplicate session ID" {
				seedSession(t, db, "duplicate-id", 1, time.Now().Add(7*24*time.Hour), nil)
			}

			err := repo.Create(context.Background(), tt.session)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validateFunc != nil {
					tt.validateFunc(t, db, tt.session)
				}
			}
		})
	}
}

func TestSessionMySQL_FindByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		sessionID   string
		setupFunc   func(t *testing.T, db *gorm.DB)
		wantErr     bool
		expectedErr error
	}{
		{
			name:      "success: find session by ID",
			sessionID: "find-session-id",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedSession(t, db, "find-session-id", 1, time.Now().Add(7*24*time.Hour), nil)
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

			db := setupSessionTestDB(t)
			repo := NewSessionMySQL(db)

			if tt.setupFunc != nil {
				tt.setupFunc(t, db)
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

func TestSessionMySQL_FindByUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		userID        uint
		setupFunc     func(t *testing.T, db *gorm.DB)
		expectedCount int
	}{
		{
			name:   "success: find active sessions for user",
			userID: 1,
			setupFunc: func(t *testing.T, db *gorm.DB) {
				// Active sessions
				seedSession(t, db, "session-1", 1, time.Now().Add(7*24*time.Hour), nil)
				seedSession(t, db, "session-2", 1, time.Now().Add(7*24*time.Hour), nil)
				// Expired session (should not be returned)
				seedSession(t, db, "session-expired", 1, time.Now().Add(-1*time.Hour), nil)
				// Revoked session (should not be returned)
				now := time.Now()
				seedSession(t, db, "session-revoked", 1, time.Now().Add(7*24*time.Hour), &now)
				// Other user's session (should not be returned)
				seedSession(t, db, "session-other", 2, time.Now().Add(7*24*time.Hour), nil)
			},
			expectedCount: 2,
		},
		{
			name:          "success: no sessions for user",
			userID:        999,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupSessionTestDB(t)
			repo := NewSessionMySQL(db)

			if tt.setupFunc != nil {
				tt.setupFunc(t, db)
			}

			sessions, err := repo.FindByUserID(context.Background(), tt.userID)

			assert.NoError(t, err)
			assert.Len(t, sessions, tt.expectedCount)
		})
	}
}

func TestSessionMySQL_Revoke(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		sessionID   string
		setupFunc   func(t *testing.T, db *gorm.DB)
		wantErr     bool
		expectedErr error
	}{
		{
			name:      "success: revoke session",
			sessionID: "revoke-session-id",
			setupFunc: func(t *testing.T, db *gorm.DB) {
				seedSession(t, db, "revoke-session-id", 1, time.Now().Add(7*24*time.Hour), nil)
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

			db := setupSessionTestDB(t)
			repo := NewSessionMySQL(db)

			if tt.setupFunc != nil {
				tt.setupFunc(t, db)
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
				var found SessionModel
				db.Where("id = ?", tt.sessionID).First(&found)
				assert.NotNil(t, found.RevokedAt)
			}
		})
	}
}

func TestSessionMySQL_RevokeAllByUserID(t *testing.T) {
	t.Parallel()

	db := setupSessionTestDB(t)
	repo := NewSessionMySQL(db)

	// Setup: create multiple sessions for user 1
	seedSession(t, db, "session-1", 1, time.Now().Add(7*24*time.Hour), nil)
	seedSession(t, db, "session-2", 1, time.Now().Add(7*24*time.Hour), nil)
	seedSession(t, db, "session-other", 2, time.Now().Add(7*24*time.Hour), nil)

	err := repo.RevokeAllByUserID(context.Background(), 1)
	assert.NoError(t, err)

	// Verify user 1's sessions are revoked
	var user1Sessions []SessionModel
	db.Where("user_id = ?", 1).Find(&user1Sessions)
	for _, s := range user1Sessions {
		assert.NotNil(t, s.RevokedAt, "session %s should be revoked", s.ID)
	}

	// Verify user 2's session is not revoked
	var user2Session SessionModel
	db.Where("user_id = ?", 2).First(&user2Session)
	assert.Nil(t, user2Session.RevokedAt, "user 2's session should not be revoked")
}

func TestSessionMySQL_CountByUserID(t *testing.T) {
	t.Parallel()

	db := setupSessionTestDB(t)
	repo := NewSessionMySQL(db)

	// Setup: create sessions with various states
	seedSession(t, db, "active-1", 1, time.Now().Add(7*24*time.Hour), nil)
	seedSession(t, db, "active-2", 1, time.Now().Add(7*24*time.Hour), nil)
	seedSession(t, db, "expired", 1, time.Now().Add(-1*time.Hour), nil)
	now := time.Now()
	seedSession(t, db, "revoked", 1, time.Now().Add(7*24*time.Hour), &now)

	count, err := repo.CountByUserID(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, int64(2), count, "should only count active sessions")
}

func TestSessionMySQL_DeleteOldestByUserID(t *testing.T) {
	t.Parallel()

	db := setupSessionTestDB(t)
	repo := NewSessionMySQL(db)

	// Setup: create sessions with different creation times
	now := time.Now()
	oldSession := &SessionModel{
		ID:        "oldest-session",
		UserID:    1,
		UserAgent: "test",
		IPAddress: "127.0.0.1",
		CreatedAt: now.Add(-2 * time.Hour),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	db.Create(oldSession)

	newSession := &SessionModel{
		ID:        "newest-session",
		UserID:    1,
		UserAgent: "test",
		IPAddress: "127.0.0.1",
		CreatedAt: now.Add(-1 * time.Hour),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	db.Create(newSession)

	err := repo.DeleteOldestByUserID(context.Background(), 1)
	assert.NoError(t, err)

	// Verify oldest session is deleted
	var count int64
	db.Model(&SessionModel{}).Where("id = ?", "oldest-session").Count(&count)
	assert.Equal(t, int64(0), count, "oldest session should be deleted")

	// Verify newest session still exists
	db.Model(&SessionModel{}).Where("id = ?", "newest-session").Count(&count)
	assert.Equal(t, int64(1), count, "newest session should still exist")
}

func TestSessionMySQL_DeleteExpired(t *testing.T) {
	t.Parallel()

	db := setupSessionTestDB(t)
	repo := NewSessionMySQL(db)

	// Setup: create expired and active sessions
	seedSession(t, db, "expired-1", 1, time.Now().Add(-1*time.Hour), nil)
	seedSession(t, db, "expired-2", 1, time.Now().Add(-2*time.Hour), nil)
	seedSession(t, db, "active", 1, time.Now().Add(7*24*time.Hour), nil)

	deleted, err := repo.DeleteExpired(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, int64(2), deleted, "should delete 2 expired sessions")

	// Verify only active session remains
	var count int64
	db.Model(&SessionModel{}).Count(&count)
	assert.Equal(t, int64(1), count, "only active session should remain")
}
