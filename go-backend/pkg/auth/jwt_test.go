package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTManager(t *testing.T) {
	secret := "test-secret-key"
	expiry := time.Hour
	jwtManager := NewJWTManager(secret, expiry)

	userID := int64(12345)
	username := "testuser"

	t.Run("GenerateToken", func(t *testing.T) {
		token, err := jwtManager.GenerateToken(userID, username)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("VerifyToken_Success", func(t *testing.T) {
		token, err := jwtManager.GenerateToken(userID, username)
		require.NoError(t, err)

		claims, err := jwtManager.VerifyToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, username, claims.Username)
		assert.NotEmpty(t, claims.TokenID)
	})

	t.Run("VerifyToken_InvalidToken", func(t *testing.T) {
		invalidToken := "invalid.token.here"

		_, err := jwtManager.VerifyToken(invalidToken)
		assert.Error(t, err)
	})

	t.Run("GenerateTokenPair", func(t *testing.T) {
		tokenPair, err := jwtManager.GenerateTokenPair(userID, username)
		require.NoError(t, err)

		assert.NotEmpty(t, tokenPair.AccessToken)
		assert.NotEmpty(t, tokenPair.RefreshToken)
		assert.True(t, tokenPair.AccessExpiry.After(time.Now()))
		assert.True(t, tokenPair.RefreshExpiry.After(time.Now()))
	})

	t.Run("VerifyRefreshToken_Success", func(t *testing.T) {
		tokenPair, err := jwtManager.GenerateTokenPair(userID, username)
		require.NoError(t, err)

		claims, err := jwtManager.VerifyRefreshToken(tokenPair.RefreshToken)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, username, claims.Username)
	})

	t.Run("RefreshTokenPair", func(t *testing.T) {
		tokenPair, err := jwtManager.GenerateTokenPair(userID, username)
		require.NoError(t, err)

		newTokenPair, err := jwtManager.RefreshTokenPair(tokenPair.RefreshToken)
		require.NoError(t, err)

		assert.NotEmpty(t, newTokenPair.AccessToken)
		assert.NotEmpty(t, newTokenPair.RefreshToken)
		assert.NotEqual(t, tokenPair.AccessToken, newTokenPair.AccessToken)
		assert.NotEqual(t, tokenPair.RefreshToken, newTokenPair.RefreshToken)
	})

	t.Run("RevokeToken", func(t *testing.T) {
		token, err := jwtManager.GenerateToken(userID, username)
		require.NoError(t, err)

		err = jwtManager.RevokeToken(token)
		require.NoError(t, err)

		// 撤销后验证应该失败
		_, err = jwtManager.VerifyToken(token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "blacklisted")
	})

	t.Run("RefreshToken_Backward_Compatibility", func(t *testing.T) {
		token, err := jwtManager.GenerateToken(userID, username)
		require.NoError(t, err)

		newToken, err := jwtManager.RefreshToken(token)
		require.NoError(t, err)
		assert.NotEmpty(t, newToken)
		assert.NotEqual(t, token, newToken)
	})

	t.Run("GetTokenID", func(t *testing.T) {
		token, err := jwtManager.GenerateToken(userID, username)
		require.NoError(t, err)

		tokenID, err := jwtManager.GetTokenID(token)
		require.NoError(t, err)
		assert.NotEmpty(t, tokenID)
	})
}

func TestTokenBlacklistIntegration(t *testing.T) {
	secret := "test-secret-key"
	expiry := time.Hour
	jwtManager := NewJWTManager(secret, expiry)

	// 设置自定义黑名单
	blacklist := NewMemoryTokenBlacklist()
	jwtManager.SetTokenBlacklist(blacklist)

	userID := int64(12345)
	username := "testuser"

	t.Run("TokenBlacklist_AddAndCheck", func(t *testing.T) {
		token, err := jwtManager.GenerateToken(userID, username)
		require.NoError(t, err)

		tokenID, err := jwtManager.GetTokenID(token)
		require.NoError(t, err)

		// 添加到黑名单
		err = blacklist.Add(tokenID, time.Hour)
		require.NoError(t, err)

		// 验证应该失败
		_, err = jwtManager.VerifyToken(token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "blacklisted")
	})
}
