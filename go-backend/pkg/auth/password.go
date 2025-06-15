package auth

import (
	"go-backend/pkg/security"
)

type PasswordManager struct {
}

func NewPasswordManager() *PasswordManager {
	return &PasswordManager{}
}

// HashPassword 生成密码哈希和盐值
func (p *PasswordManager) HashPassword(password string) (string, string, error) {
	// 先验证密码强度
	if err := security.ValidatePassword(password); err != nil {
		return "", "", err
	}

	// 使用Argon2id加密
	return security.HashPassword(password)
}

// VerifyPassword 验证密码
func (p *PasswordManager) VerifyPassword(password, hash, salt string) (bool, error) {
	return security.VerifyPassword(password, hash, salt)
}
