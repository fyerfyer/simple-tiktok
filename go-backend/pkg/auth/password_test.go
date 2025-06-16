package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordManager(t *testing.T) {
	pm := NewPasswordManager()

	t.Run("HashPassword", func(t *testing.T) {
		password := "TestPassword123"

		hash, salt, err := pm.HashPassword(password)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEmpty(t, salt)
		assert.NotEqual(t, password, hash)
	})

	t.Run("VerifyPassword_Success", func(t *testing.T) {
		password := "TestPassword123"
		hash, salt, err := pm.HashPassword(password)
		require.NoError(t, err)

		valid, err := pm.VerifyPassword(password, hash, salt)
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("VerifyPassword_WrongPassword", func(t *testing.T) {
		password := "TestPassword123"
		wrongPassword := "WrongPassword123"
		hash, salt, err := pm.HashPassword(password)
		require.NoError(t, err)

		valid, err := pm.VerifyPassword(wrongPassword, hash, salt)
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("HashPassword_WeakPassword", func(t *testing.T) {
		weakPassword := "123"

		_, _, err := pm.HashPassword(weakPassword)
		assert.Error(t, err)
	})

	t.Run("VerifyPassword_InvalidSalt", func(t *testing.T) {
		password := "TestPassword123"
		hash, _, err := pm.HashPassword(password)
		require.NoError(t, err)

		valid, err := pm.VerifyPassword(password, hash, "invalid_salt")
		assert.Error(t, err)
		assert.False(t, valid)
	})

	t.Run("VerifyPassword_InvalidHash", func(t *testing.T) {
		password := "TestPassword123"
		_, salt, err := pm.HashPassword(password)
		require.NoError(t, err)

		valid, err := pm.VerifyPassword(password, "invalid_hash", salt)
		assert.Error(t, err)
		assert.False(t, valid)
	})
}
