package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"golang.org/x/crypto/pbkdf2"
)

const (
	defaultIterations = 10000
	defaultKeyLength  = 32
	defaultSaltLength = 32
)

type PasswordManager struct {
	iterations int
	keyLength  int
	saltLength int
}

func NewPasswordManager() *PasswordManager {
	return &PasswordManager{
		iterations: defaultIterations,
		keyLength:  defaultKeyLength,
		saltLength: defaultSaltLength,
	}
}

// 生成密码哈希和盐值
func (p *PasswordManager) HashPassword(password string) (string, string, error) {
	salt := make([]byte, p.saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", "", err
	}

	hash := pbkdf2.Key([]byte(password), salt, p.iterations, p.keyLength, sha256.New)
	return base64.StdEncoding.EncodeToString(hash),
		base64.StdEncoding.EncodeToString(salt), nil
}

// 验证密码
func (p *PasswordManager) VerifyPassword(password, hash, salt string) bool {
	saltBytes, err := base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return false
	}

	hashBytes, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		return false
	}

	computedHash := pbkdf2.Key([]byte(password), saltBytes, p.iterations, p.keyLength, sha256.New)
	return compareHashes(hashBytes, computedHash)
}

func compareHashes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	result := byte(0)
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
