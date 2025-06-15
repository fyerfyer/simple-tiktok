package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-backend/internal/domain"
	"go-backend/pkg/cache"

	"github.com/go-kratos/kratos/v2/log"
)

// AuthCache 认证缓存
type AuthCache struct {
	cache    *cache.MultiLevelCache
	strategy *cache.CacheStrategy
	log      *log.Helper
}

// NewAuthCache 创建认证缓存
func NewAuthCache(multiCache *cache.MultiLevelCache, logger log.Logger) *AuthCache {
	strategy := cache.NewCacheStrategy(multiCache)
	return &AuthCache{
		cache:    multiCache,
		strategy: strategy,
		log:      log.NewHelper(logger),
	}
}

// AddTokenToBlacklist 添加Token到黑名单
func (c *AuthCache) AddTokenToBlacklist(ctx context.Context, tokenID string, expireTime time.Duration) error {
	return c.strategy.AddTokenToBlacklist(ctx, tokenID, expireTime)
}

// IsTokenBlacklisted 检查Token是否在黑名单
func (c *AuthCache) IsTokenBlacklisted(ctx context.Context, tokenID string) bool {
	return c.strategy.IsTokenBlacklisted(ctx, tokenID)
}

// SetRefreshToken 设置Refresh Token
func (c *AuthCache) SetRefreshToken(ctx context.Context, userID int64, refreshToken string, expireTime time.Duration) error {
	key := fmt.Sprintf("refresh_token:%d", userID)

	tokenData := map[string]interface{}{
		"token":      refreshToken,
		"expires_at": time.Now().Add(expireTime).Unix(),
		"created_at": time.Now().Unix(),
	}

	data, err := json.Marshal(tokenData)
	if err != nil {
		return fmt.Errorf("marshal refresh token failed: %w", err)
	}

	return c.cache.SetString(ctx, key, string(data), expireTime)
}

// GetRefreshToken 获取Refresh Token
func (c *AuthCache) GetRefreshToken(ctx context.Context, userID int64) (string, error) {
	key := fmt.Sprintf("refresh_token:%d", userID)

	data, err := c.cache.GetString(ctx, key)
	if err != nil {
		return "", err
	}

	var tokenData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
		return "", err
	}

	token, ok := tokenData["token"].(string)
	if !ok {
		return "", fmt.Errorf("invalid refresh token data")
	}

	return token, nil
}

// DeleteRefreshToken 删除Refresh Token
func (c *AuthCache) DeleteRefreshToken(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("refresh_token:%d", userID)
	return c.cache.Delete(ctx, key)
}

// SetUserSession 设置用户会话
func (c *AuthCache) SetUserSession(ctx context.Context, session *domain.UserSession) error {
	key := fmt.Sprintf("session:%d", session.UserID)

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal user session failed: %w", err)
	}

	expireTime := time.Until(session.ExpiresAt)
	return c.cache.SetString(ctx, key, string(data), expireTime)
}

// GetUserSession 获取用户会话
func (c *AuthCache) GetUserSession(ctx context.Context, userID int64) (*domain.UserSession, error) {
	key := fmt.Sprintf("session:%d", userID)

	data, err := c.cache.GetString(ctx, key)
	if err != nil {
		return nil, err
	}

	var session domain.UserSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// DeleteUserSession 删除用户会话
func (c *AuthCache) DeleteUserSession(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("session:%d", userID)
	return c.cache.Delete(ctx, key)
}

// SetLoginAttempts 设置登录尝试次数
func (c *AuthCache) SetLoginAttempts(ctx context.Context, username string, attempts int) error {
	key := fmt.Sprintf("login_attempts:%s", username)
	return c.cache.SetString(ctx, key, fmt.Sprintf("%d", attempts), 15*time.Minute)
}

// GetLoginAttempts 获取登录尝试次数
func (c *AuthCache) GetLoginAttempts(ctx context.Context, username string) (int, error) {
	key := fmt.Sprintf("login_attempts:%s", username)

	data, err := c.cache.GetString(ctx, key)
	if err != nil {
		return 0, nil // 没有记录则返回0
	}

	var attempts int
	if _, err := fmt.Sscanf(data, "%d", &attempts); err != nil {
		return 0, err
	}

	return attempts, nil
}

// ClearLoginAttempts 清除登录尝试次数
func (c *AuthCache) ClearLoginAttempts(ctx context.Context, username string) error {
	key := fmt.Sprintf("login_attempts:%s", username)
	return c.cache.Delete(ctx, key)
}

// SetPasswordResetToken 设置密码重置Token
func (c *AuthCache) SetPasswordResetToken(ctx context.Context, email, token string) error {
	key := fmt.Sprintf("password_reset:%s", email)

	tokenData := map[string]interface{}{
		"token":      token,
		"created_at": time.Now().Unix(),
	}

	data, err := json.Marshal(tokenData)
	if err != nil {
		return fmt.Errorf("marshal password reset token failed: %w", err)
	}

	return c.cache.SetString(ctx, key, string(data), 30*time.Minute)
}

// VerifyPasswordResetToken 验证密码重置Token
func (c *AuthCache) VerifyPasswordResetToken(ctx context.Context, email, token string) (bool, error) {
	key := fmt.Sprintf("password_reset:%s", email)

	data, err := c.cache.GetString(ctx, key)
	if err != nil {
		return false, nil
	}

	var tokenData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
		return false, err
	}

	cachedToken, ok := tokenData["token"].(string)
	if !ok {
		return false, fmt.Errorf("invalid password reset token data")
	}

	return cachedToken == token, nil
}

// DeletePasswordResetToken 删除密码重置Token
func (c *AuthCache) DeletePasswordResetToken(ctx context.Context, email string) error {
	key := fmt.Sprintf("password_reset:%s", email)
	return c.cache.Delete(ctx, key)
}

// SetUserPermissions 设置用户权限缓存
func (c *AuthCache) SetUserPermissions(ctx context.Context, userID int64, permissions []string) error {
	key := fmt.Sprintf("user_permissions:%d", userID)

	data, err := json.Marshal(permissions)
	if err != nil {
		return fmt.Errorf("marshal user permissions failed: %w", err)
	}

	return c.cache.SetString(ctx, key, string(data), 60*time.Minute)
}

// GetUserPermissions 获取用户权限缓存
func (c *AuthCache) GetUserPermissions(ctx context.Context, userID int64) ([]string, error) {
	key := fmt.Sprintf("user_permissions:%d", userID)

	data, err := c.cache.GetString(ctx, key)
	if err != nil {
		return nil, err
	}

	var permissions []string
	if err := json.Unmarshal([]byte(data), &permissions); err != nil {
		return nil, err
	}

	return permissions, nil
}

// SetUserRoles 设置用户角色缓存
func (c *AuthCache) SetUserRoles(ctx context.Context, userID int64, roles []string) error {
	key := fmt.Sprintf("user_roles:%d", userID)

	data, err := json.Marshal(roles)
	if err != nil {
		return fmt.Errorf("marshal user roles failed: %w", err)
	}

	return c.cache.SetString(ctx, key, string(data), 60*time.Minute)
}

// GetUserRoles 获取用户角色缓存
func (c *AuthCache) GetUserRoles(ctx context.Context, userID int64) ([]string, error) {
	key := fmt.Sprintf("user_roles:%d", userID)

	data, err := c.cache.GetString(ctx, key)
	if err != nil {
		return nil, err
	}

	var roles []string
	if err := json.Unmarshal([]byte(data), &roles); err != nil {
		return nil, err
	}

	return roles, nil
}

// ClearUserAuth 清除用户认证相关缓存
func (c *AuthCache) ClearUserAuth(ctx context.Context, userID int64) error {
	keys := []string{
		fmt.Sprintf("session:%d", userID),
		fmt.Sprintf("refresh_token:%d", userID),
		fmt.Sprintf("user_permissions:%d", userID),
		fmt.Sprintf("user_roles:%d", userID),
	}

	for _, key := range keys {
		if err := c.cache.Delete(ctx, key); err != nil {
			c.log.WithContext(ctx).Errorf("clear user auth cache failed: %v", err)
		}
	}

	return nil
}

// SetOnlineUsers 设置在线用户列表
func (c *AuthCache) SetOnlineUsers(ctx context.Context, userIDs []int64) error {
	key := "online_users"

	data, err := json.Marshal(userIDs)
	if err != nil {
		return fmt.Errorf("marshal online users failed: %w", err)
	}

	return c.cache.SetString(ctx, key, string(data), 5*time.Minute)
}

// GetOnlineUsers 获取在线用户列表
func (c *AuthCache) GetOnlineUsers(ctx context.Context) ([]int64, error) {
	key := "online_users"

	data, err := c.cache.GetString(ctx, key)
	if err != nil {
		return nil, err
	}

	var userIDs []int64
	if err := json.Unmarshal([]byte(data), &userIDs); err != nil {
		return nil, err
	}

	return userIDs, nil
}

// AddOnlineUser 添加在线用户
func (c *AuthCache) AddOnlineUser(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("online_user:%d", userID)
	return c.cache.SetString(ctx, key, "1", 10*time.Minute)
}

// RemoveOnlineUser 移除在线用户
func (c *AuthCache) RemoveOnlineUser(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("online_user:%d", userID)
	return c.cache.Delete(ctx, key)
}

// IsUserOnline 检查用户是否在线
func (c *AuthCache) IsUserOnline(ctx context.Context, userID int64) (bool, error) {
	key := fmt.Sprintf("online_user:%d", userID)
	_, err := c.cache.GetString(ctx, key)
	if err != nil {
		return false, nil
	}
	return true, nil
}
