package domain

import "time"

// UserSession 用户会话领域模型
type UserSession struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// TokenBlacklist Token黑名单领域模型
type TokenBlacklist struct {
	ID        int64     `json:"id"`
	TokenID   string    `json:"token_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// TokenPair Token对
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// IsExpired 检查会话是否过期
func (s *UserSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsExpired 检查黑名单项是否过期
func (t *TokenBlacklist) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// Refresh 刷新会话
func (s *UserSession) Refresh(newToken string, duration time.Duration) {
	s.RefreshToken = newToken
	s.ExpiresAt = time.Now().Add(duration)
}
