package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	password := "TestPassword123!"

	hash, salt, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEmpty(t, salt)
	assert.NotEqual(t, password, hash)
	assert.NotEqual(t, password, salt)
}

func TestVerifyPassword_Success(t *testing.T) {
	password := "TestPassword123!"
	hash, salt, err := HashPassword(password)
	require.NoError(t, err)

	valid, err := VerifyPassword(password, hash, salt)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	password := "TestPassword123!"
	wrongPassword := "WrongPassword123!"
	hash, salt, err := HashPassword(password)
	require.NoError(t, err)

	valid, err := VerifyPassword(wrongPassword, hash, salt)
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestVerifyPassword_InvalidSalt(t *testing.T) {
	password := "TestPassword123!"
	hash, _, err := HashPassword(password)
	require.NoError(t, err)

	valid, err := VerifyPassword(password, hash, "invalid_salt")
	assert.Error(t, err)
	assert.False(t, valid)
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	password := "TestPassword123!"
	_, salt, err := HashPassword(password)
	require.NoError(t, err)

	valid, err := VerifyPassword(password, "invalid_hash", salt)
	assert.Error(t, err)
	assert.False(t, valid)
}

func TestGenerateRandomString(t *testing.T) {
	length := 16
	str1, err := GenerateRandomString(length)
	require.NoError(t, err)
	assert.Len(t, str1, length)

	str2, err := GenerateRandomString(length)
	require.NoError(t, err)
	assert.Len(t, str2, length)

	// 生成的字符串应该不同
	assert.NotEqual(t, str1, str2)
}

func TestGenerateTokenID(t *testing.T) {
	token1, err := GenerateTokenID()
	require.NoError(t, err)
	assert.NotEmpty(t, token1)

	token2, err := GenerateTokenID()
	require.NoError(t, err)
	assert.NotEmpty(t, token2)

	// 生成的Token ID应该不同
	assert.NotEqual(t, token1, token2)
}

func TestHashPasswordConsistency(t *testing.T) {
	password := "SamePassword123!"

	hash1, salt1, err := HashPassword(password)
	require.NoError(t, err)

	hash2, salt2, err := HashPassword(password)
	require.NoError(t, err)

	// 相同密码的hash和salt应该不同（因为盐值随机）
	assert.NotEqual(t, hash1, hash2)
	assert.NotEqual(t, salt1, salt2)

	// 但都应该能验证通过
	valid1, err := VerifyPassword(password, hash1, salt1)
	require.NoError(t, err)
	assert.True(t, valid1)

	valid2, err := VerifyPassword(password, hash2, salt2)
	require.NoError(t, err)
	assert.True(t, valid2)
}

func TestPasswordSecurity(t *testing.T) {
	password := "SecurePassword123!"
	hash, salt, err := HashPassword(password)
	require.NoError(t, err)

	// 确保hash和salt不包含原始密码
	assert.NotContains(t, hash, password)
	assert.NotContains(t, salt, password)
	assert.NotContains(t, hash, "SecurePassword")
	assert.NotContains(t, salt, "SecurePassword")
}
