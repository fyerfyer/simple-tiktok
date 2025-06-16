package auth

import (
	"testing"
	"time"

	"go-backend/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemorySessionManager(t *testing.T) {
	manager := NewMemorySessionManager()
	defer manager.cleanup()

	userID := int64(12345)
	refreshToken := "test-refresh-token"
	expiry := time.Hour

	t.Run("CreateSession", func(t *testing.T) {
		session, err := manager.CreateSession(userID, refreshToken, expiry)
		require.NoError(t, err)
		assert.Equal(t, userID, session.UserID)
		assert.Equal(t, refreshToken, session.RefreshToken)
		assert.True(t, session.ExpiresAt.After(time.Now()))
	})

	t.Run("GetSession_Success", func(t *testing.T) {
		_, err := manager.CreateSession(userID, refreshToken, expiry)
		require.NoError(t, err)

		session, err := manager.GetSession(userID)
		require.NoError(t, err)
		assert.Equal(t, userID, session.UserID)
		assert.Equal(t, refreshToken, session.RefreshToken)
	})

	t.Run("GetSession_NotFound", func(t *testing.T) {
		nonExistentUserID := int64(99999)

		_, err := manager.GetSession(nonExistentUserID)
		assert.Error(t, err)
		assert.Equal(t, ErrSessionNotFound, err)
	})

	t.Run("UpdateSession", func(t *testing.T) {
		_, err := manager.CreateSession(userID, refreshToken, expiry)
		require.NoError(t, err)

		newRefreshToken := "new-refresh-token"
		err = manager.UpdateSession(userID, newRefreshToken, expiry)
		require.NoError(t, err)

		session, err := manager.GetSession(userID)
		require.NoError(t, err)
		assert.Equal(t, newRefreshToken, session.RefreshToken)
	})

	t.Run("DeleteSession", func(t *testing.T) {
		_, err := manager.CreateSession(userID, refreshToken, expiry)
		require.NoError(t, err)

		err = manager.DeleteSession(userID)
		require.NoError(t, err)

		_, err = manager.GetSession(userID)
		assert.Error(t, err)
		assert.Equal(t, ErrSessionNotFound, err)
	})

	t.Run("IsSessionValid_Success", func(t *testing.T) {
		_, err := manager.CreateSession(userID, refreshToken, expiry)
		require.NoError(t, err)

		isValid := manager.IsSessionValid(userID, refreshToken)
		assert.True(t, isValid)
	})

	t.Run("IsSessionValid_WrongToken", func(t *testing.T) {
		_, err := manager.CreateSession(userID, refreshToken, expiry)
		require.NoError(t, err)

		isValid := manager.IsSessionValid(userID, "wrong-token")
		assert.False(t, isValid)
	})

	t.Run("Session_Expiry", func(t *testing.T) {
		shortExpiry := 100 * time.Millisecond
		_, err := manager.CreateSession(userID, refreshToken, shortExpiry)
		require.NoError(t, err)

		// 等待过期
		time.Sleep(200 * time.Millisecond)

		_, err = manager.GetSession(userID)
		assert.Error(t, err)
		assert.Equal(t, ErrSessionExpired, err)
	})

	t.Run("GetAllSessions", func(t *testing.T) {
		user1 := int64(1001)
		user2 := int64(1002)

		_, err := manager.CreateSession(user1, "token1", expiry)
		require.NoError(t, err)

		_, err = manager.CreateSession(user2, "token2", expiry)
		require.NoError(t, err)

		sessions := manager.GetAllSessions()
		assert.Len(t, sessions, 2)
		assert.Contains(t, []int64{user1, user2}, sessions[user1].UserID)
		assert.Contains(t, []int64{user1, user2}, sessions[user2].UserID)
	})
}

func TestSessionDomainMethods(t *testing.T) {
	t.Run("Session_IsExpired", func(t *testing.T) {
		session := &domain.UserSession{
			UserID:       12345,
			RefreshToken: "test-token",
			ExpiresAt:    time.Now().Add(-time.Hour), // 已过期
			CreatedAt:    time.Now().Add(-2 * time.Hour),
		}

		assert.True(t, session.IsExpired())
	})

	t.Run("Session_Refresh", func(t *testing.T) {
		session := &domain.UserSession{
			UserID:       12345,
			RefreshToken: "old-token",
			ExpiresAt:    time.Now().Add(time.Hour),
			CreatedAt:    time.Now(),
		}

		newToken := "new-token"
		newExpiry := 2 * time.Hour
		oldExpiresAt := session.ExpiresAt

		session.Refresh(newToken, newExpiry)

		assert.Equal(t, newToken, session.RefreshToken)
		assert.True(t, session.ExpiresAt.After(oldExpiresAt))
	})
}
