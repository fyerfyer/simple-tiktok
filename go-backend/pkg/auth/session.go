package auth

import (
	"sync"
	"time"

	"go-backend/internal/domain"
)

// SessionManager 会话管理器接口
type SessionManager interface {
	CreateSession(userID int64, refreshToken string, expiry time.Duration) (*domain.UserSession, error)
	GetSession(userID int64) (*domain.UserSession, error)
	UpdateSession(userID int64, newRefreshToken string, expiry time.Duration) error
	DeleteSession(userID int64) error
	IsSessionValid(userID int64, refreshToken string) bool
	CleanupExpiredSessions() error
}

// MemorySessionManager 内存会话管理器
type MemorySessionManager struct {
	sessions map[int64]*domain.UserSession
	mutex    sync.RWMutex
}

// NewMemorySessionManager 创建内存会话管理器
func NewMemorySessionManager() *MemorySessionManager {
	manager := &MemorySessionManager{
		sessions: make(map[int64]*domain.UserSession),
	}

	// 启动清理goroutine
	go manager.cleanup()

	return manager
}

// CreateSession 创建会话
func (s *MemorySessionManager) CreateSession(userID int64, refreshToken string, expiry time.Duration) (*domain.UserSession, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	session := &domain.UserSession{
		UserID:       userID,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(expiry),
		CreatedAt:    time.Now(),
	}

	s.sessions[userID] = session
	return session, nil
}

// GetSession 获取会话
func (s *MemorySessionManager) GetSession(userID int64) (*domain.UserSession, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	session, exists := s.sessions[userID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	// 检查会话是否过期
	if session.IsExpired() {
		s.mutex.RUnlock()
		s.mutex.Lock()
		delete(s.sessions, userID)
		s.mutex.Unlock()
		s.mutex.RLock()
		return nil, ErrSessionExpired
	}

	return session, nil
}

// UpdateSession 更新会话
func (s *MemorySessionManager) UpdateSession(userID int64, newRefreshToken string, expiry time.Duration) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, exists := s.sessions[userID]
	if !exists {
		return ErrSessionNotFound
	}

	session.Refresh(newRefreshToken, expiry)
	return nil
}

// DeleteSession 删除会话
func (s *MemorySessionManager) DeleteSession(userID int64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.sessions, userID)
	return nil
}

// IsSessionValid 检查会话是否有效
func (s *MemorySessionManager) IsSessionValid(userID int64, refreshToken string) bool {
	session, err := s.GetSession(userID)
	if err != nil {
		return false
	}

	return session.RefreshToken == refreshToken && !session.IsExpired()
}

// CleanupExpiredSessions 清理过期会话
func (s *MemorySessionManager) CleanupExpiredSessions() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	for userID, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, userID)
		}
	}

	return nil
}

// cleanup 定期清理过期会话
func (s *MemorySessionManager) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.CleanupExpiredSessions()
	}
}

// GetAllSessions 获取所有会话 (调试用)
func (s *MemorySessionManager) GetAllSessions() map[int64]*domain.UserSession {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make(map[int64]*domain.UserSession)
	for k, v := range s.sessions {
		result[k] = v
	}
	return result
}

// 会话相关错误
var (
	ErrSessionNotFound = NewAuthError("session not found")
	ErrSessionExpired  = NewAuthError("session expired")
)

// AuthError 认证错误
type AuthError struct {
	message string
}

func NewAuthError(message string) *AuthError {
	return &AuthError{message: message}
}

func (e *AuthError) Error() string {
	return e.message
}
