package data

import (
	"context"
	"fmt"
	"time"

	"go-backend/internal/data/cache"
	"go-backend/internal/domain"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// UserSession 用户会话模型
type UserSession struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int64     `gorm:"not null;index" json:"user_id"`
	RefreshToken string    `gorm:"uniqueIndex;size:255;not null" json:"refresh_token"`
	ExpiresAt    time.Time `gorm:"not null;index" json:"expires_at"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (UserSession) TableName() string {
	return "user_sessions"
}

// TokenBlacklist Token黑名单模型
type TokenBlacklist struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TokenID   string    `gorm:"uniqueIndex;size:255;not null" json:"token_id"`
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (TokenBlacklist) TableName() string {
	return "token_blacklist"
}

// SessionRepo 会话仓储实现 - 实现 biz.AuthRepo 接口
type SessionRepo struct {
	data      *Data
	authCache *cache.AuthCache
	log       *log.Helper
}

// NewSessionRepo 创建会话仓储
func NewSessionRepo(data *Data, authCache *cache.AuthCache, logger log.Logger) *SessionRepo {
	return &SessionRepo{
		data:      data,
		authCache: authCache,
		log:       log.NewHelper(logger),
	}
}

// 实现 biz.AuthRepo 接口的所有方法
func (r *SessionRepo) CreateSession(ctx context.Context, session *domain.UserSession) error {
	s := &UserSession{
		UserID:       session.UserID,
		RefreshToken: session.RefreshToken,
		ExpiresAt:    session.ExpiresAt,
	}

	if err := r.data.db.WithContext(ctx).Create(s).Error; err != nil {
		return err
	}

	session.ID = s.ID
	session.CreatedAt = s.CreatedAt

	// 设置缓存
	r.authCache.SetUserSession(ctx, session)

	return nil
}

func (r *SessionRepo) GetSession(ctx context.Context, userID int64) (*domain.UserSession, error) {
	// 先从缓存获取
	if session, err := r.authCache.GetUserSession(ctx, userID); err == nil {
		if !session.IsExpired() {
			return session, nil
		}
		// 过期了，删除缓存
		r.authCache.DeleteUserSession(ctx, userID)
	}

	var s UserSession
	if err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND expires_at > ?", userID, time.Now()).
		First(&s).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("session not found")
		}
		return nil, err
	}

	session := r.convertToSession(&s)
	r.authCache.SetUserSession(ctx, session)

	return session, nil
}

func (r *SessionRepo) GetSessionByToken(ctx context.Context, refreshToken string) (*domain.UserSession, error) {
	var s UserSession
	if err := r.data.db.WithContext(ctx).
		Where("refresh_token = ? AND expires_at > ?", refreshToken, time.Now()).
		First(&s).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("session not found")
		}
		return nil, err
	}

	return r.convertToSession(&s), nil
}

func (r *SessionRepo) UpdateSession(ctx context.Context, userID int64, newRefreshToken string, expiry time.Duration) error {
	expiresAt := time.Now().Add(expiry)

	if err := r.data.db.WithContext(ctx).Model(&UserSession{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"refresh_token": newRefreshToken,
			"expires_at":    expiresAt,
		}).Error; err != nil {
		return err
	}

	r.authCache.DeleteUserSession(ctx, userID)
	return nil
}

func (r *SessionRepo) DeleteSession(ctx context.Context, userID int64) error {
	if err := r.data.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&UserSession{}).Error; err != nil {
		return err
	}

	r.authCache.DeleteUserSession(ctx, userID)
	return nil
}

func (r *SessionRepo) AddTokenToBlacklist(ctx context.Context, tokenID string, expiresAt time.Time) error {
	token := &TokenBlacklist{
		TokenID:   tokenID,
		ExpiresAt: expiresAt,
	}

	if err := r.data.db.WithContext(ctx).Create(token).Error; err != nil {
		return err
	}

	expiry := time.Until(expiresAt)
	r.authCache.AddTokenToBlacklist(ctx, tokenID, expiry)

	return nil
}

func (r *SessionRepo) IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	if r.authCache.IsTokenBlacklisted(ctx, tokenID) {
		return true, nil
	}

	var count int64
	err := r.data.db.WithContext(ctx).Model(&TokenBlacklist{}).
		Where("token_id = ? AND expires_at > ?", tokenID, time.Now()).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	isBlacklisted := count > 0
	if isBlacklisted {
		r.authCache.AddTokenToBlacklist(ctx, tokenID, time.Hour)
	}

	return isBlacklisted, nil
}

func (r *SessionRepo) convertToSession(s *UserSession) *domain.UserSession {
	return &domain.UserSession{
		ID:           s.ID,
		UserID:       s.UserID,
		RefreshToken: s.RefreshToken,
		ExpiresAt:    s.ExpiresAt,
		CreatedAt:    s.CreatedAt,
	}
}
