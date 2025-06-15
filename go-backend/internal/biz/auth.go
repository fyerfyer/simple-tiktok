package biz

import (
	"context"
	"time"

	"go-backend/internal/domain"
	"go-backend/pkg/auth"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var ErrSessionExpired = errors.GatewayTimeout("SESSION_EXPIRED", "session expired")

// AuthRepo 认证仓储接口
type AuthRepo interface {
	CreateSession(ctx context.Context, session *domain.UserSession) error
	GetSession(ctx context.Context, userID int64) (*domain.UserSession, error)
	GetSessionByToken(ctx context.Context, refreshToken string) (*domain.UserSession, error)
	UpdateSession(ctx context.Context, userID int64, newRefreshToken string, expiry time.Duration) error
	DeleteSession(ctx context.Context, userID int64) error
	AddTokenToBlacklist(ctx context.Context, tokenID string, expiresAt time.Time) error
	IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error)
}

// AuthUsecase 认证用例
type AuthUsecase struct {
	repo       AuthRepo
	userRepo   UserRepo
	jwtManager *auth.JWTManager
	sessionMgr auth.SessionManager
	log        *log.Helper
}

// NewAuthUsecase 创建认证用例
func NewAuthUsecase(
	repo AuthRepo,
	userRepo UserRepo,
	jwtManager *auth.JWTManager,
	sessionMgr auth.SessionManager,
	logger log.Logger,
) *AuthUsecase {
	return &AuthUsecase{
		repo:       repo,
		userRepo:   userRepo,
		jwtManager: jwtManager,
		sessionMgr: sessionMgr,
		log:        log.NewHelper(logger),
	}
}

// LoginWithToken 使用双Token机制登录
func (uc *AuthUsecase) LoginWithToken(ctx context.Context, username, password string) (*auth.TokenPair, *User, error) {
	uc.log.WithContext(ctx).Infof("Login with token: %s", username)

	// 验证用户名和密码
	user, err := uc.userRepo.VerifyPassword(ctx, username, password)
	if err != nil {
		return nil, nil, err
	}

	// 生成Token对
	tokenPair, err := uc.jwtManager.GenerateTokenPair(user.ID, user.Username)
	if err != nil {
		return nil, nil, err
	}

	// 创建会话
	session := &domain.UserSession{
		UserID:       user.ID,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.RefreshExpiry,
	}

	// 删除旧会话并创建新会话
	uc.repo.DeleteSession(ctx, user.ID)
	if err := uc.repo.CreateSession(ctx, session); err != nil {
		uc.log.WithContext(ctx).Errorf("create session failed: %v", err)
	}

	// 更新登录时间
	now := time.Now()
	user.LastLoginAt = &now
	uc.userRepo.UpdateUser(ctx, user)

	return tokenPair, user, nil
}

// RefreshToken 刷新Token
func (uc *AuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	uc.log.WithContext(ctx).Info("Refresh token")

	// 验证Refresh Token
	claims, err := uc.jwtManager.VerifyRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// 检查会话是否存在
	session, err := uc.repo.GetSessionByToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	// 生成新的Token对
	newTokenPair, err := uc.jwtManager.GenerateTokenPair(claims.UserID, claims.Username)
	if err != nil {
		return nil, err
	}

	// 将旧的Refresh Token加入黑名单
	uc.repo.AddTokenToBlacklist(ctx, claims.TokenID, time.Unix(claims.ExpiresAt.Unix(), 0))

	// 更新会话
	err = uc.repo.UpdateSession(ctx, session.UserID, newTokenPair.RefreshToken, 7*24*time.Hour)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("update session failed: %v", err)
	}

	return newTokenPair, nil
}

// Logout 登出
func (uc *AuthUsecase) Logout(ctx context.Context, userID int64, accessToken, refreshToken string) error {
	uc.log.WithContext(ctx).Infof("Logout user: %d", userID)

	// 获取Token ID并加入黑名单
	if accessTokenID, err := uc.jwtManager.GetTokenID(accessToken); err == nil {
		claims, _ := uc.jwtManager.VerifyToken(accessToken)
		if claims != nil {
			expiry := time.Until(time.Unix(claims.ExpiresAt.Unix(), 0))
			uc.repo.AddTokenToBlacklist(ctx, accessTokenID, time.Now().Add(expiry))
		}
	}

	// 撤销Refresh Token
	if refreshToken != "" {
		uc.jwtManager.RevokeRefreshToken(refreshToken)
	}

	// 删除会话
	return uc.repo.DeleteSession(ctx, userID)
}

// VerifyToken 验证Token
func (uc *AuthUsecase) VerifyToken(ctx context.Context, token string) (*auth.Claims, error) {
	return uc.jwtManager.VerifyToken(token)
}

// RevokeToken 撤销Token
func (uc *AuthUsecase) RevokeToken(ctx context.Context, token string) error {
	uc.log.WithContext(ctx).Info("Revoke token")
	return uc.jwtManager.RevokeToken(token)
}

// RevokeAllUserTokens 撤销用户所有Token
func (uc *AuthUsecase) RevokeAllUserTokens(ctx context.Context, userID int64) error {
	uc.log.WithContext(ctx).Infof("Revoke all tokens for user: %d", userID)

	// 删除用户会话
	return uc.repo.DeleteSession(ctx, userID)
}

// CheckTokenBlacklist 检查Token是否在黑名单
func (uc *AuthUsecase) CheckTokenBlacklist(ctx context.Context, tokenID string) (bool, error) {
	return uc.repo.IsTokenBlacklisted(ctx, tokenID)
}

// GetUserSession 获取用户会话
func (uc *AuthUsecase) GetUserSession(ctx context.Context, userID int64) (*domain.UserSession, error) {
	return uc.repo.GetSession(ctx, userID)
}

// ValidateSession 验证会话有效性
func (uc *AuthUsecase) ValidateSession(ctx context.Context, userID int64, refreshToken string) (bool, error) {
	session, err := uc.repo.GetSession(ctx, userID)
	if err != nil {
		return false, err
	}

	if session.RefreshToken != refreshToken {
		return false, nil
	}

	return !session.IsExpired(), nil
}

// CleanupExpiredSessions 清理过期会话
func (uc *AuthUsecase) CleanupExpiredSessions(ctx context.Context) error {
	uc.log.WithContext(ctx).Info("Cleanup expired sessions")
	// TODO: 这里可以实现定期清理逻辑
	return nil
}
