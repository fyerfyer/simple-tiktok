package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// Argon2 参数配置
const (
	saltLength = 16
	keyLength  = 32
	time       = 1
	memory     = 64 * 1024
	threads    = 4
)

// HashPassword 使用Argon2id加密密码
func HashPassword(password string) (string, string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", "", err
	}

	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLength)

	saltStr := base64.RawStdEncoding.EncodeToString(salt)
	hashStr := base64.RawStdEncoding.EncodeToString(hash)

	return hashStr, saltStr, nil
}

// VerifyPassword 验证密码
func VerifyPassword(password, hash, salt string) (bool, error) {
	saltBytes, err := base64.RawStdEncoding.DecodeString(salt)
	if err != nil {
		return false, err
	}

	hashBytes, err := base64.RawStdEncoding.DecodeString(hash)
	if err != nil {
		return false, err
	}

	comparisonHash := argon2.IDKey([]byte(password), saltBytes, time, memory, threads, keyLength)

	return subtle.ConstantTimeCompare(hashBytes, comparisonHash) == 1, nil
}

// GenerateRandomString 生成随机字符串
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// GenerateToken 生成Token ID
func GenerateTokenID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate token id: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
