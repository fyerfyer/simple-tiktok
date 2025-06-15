package service

import (
	"context"

	"go-backend/internal/biz"
	"go-backend/pkg/auth"

	"github.com/go-kratos/kratos/v2/log"
)

// AuthService 认证服务
type AuthService struct {
	authUc     *biz.AuthUsecase
	jwtManager *auth.JWTManager
	log        *log.Helper
}

// NewAuthService 创建认证服务
func NewAuthService(
	authUc *biz.AuthUsecase,
	jwtManager *auth.JWTManager,
	logger log.Logger,
) *AuthService {
	return &AuthService{
		authUc:     authUc,
		jwtManager: jwtManager,
		log:        log.NewHelper(logger),
	}
}

// RefreshToken 刷新Token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	s.log.WithContext(ctx).Info("Refresh token request")

	tokenPair, err := s.authUc.RefreshToken(ctx, refreshToken)
	if err != nil {
		s.log.WithContext(ctx).Errorf("refresh token failed: %v", err)
		return nil, err
	}

	return tokenPair, nil
}

// Logout 用户登出
func (s *AuthService) Logout(ctx context.Context, userID int64, accessToken, refreshToken string) error {
	s.log.WithContext(ctx).Infof("Logout user: %d", userID)

	err := s.authUc.Logout(ctx, userID, accessToken, refreshToken)
	if err != nil {
		s.log.WithContext(ctx).Errorf("logout failed: %v", err)
		return err
	}

	return nil
}

// RevokeToken 撤销Token
func (s *AuthService) RevokeToken(ctx context.Context, token string) error {
	s.log.WithContext(ctx).Info("Revoke token request")

	err := s.authUc.RevokeToken(ctx, token)
	if err != nil {
		s.log.WithContext(ctx).Errorf("revoke token failed: %v", err)
		return err
	}

	return nil
}

// VerifyTokenInternal 内部Token验证
func (s *AuthService) VerifyTokenInternal(ctx context.Context, token string) (*auth.Claims, error) {
	return s.authUc.VerifyToken(ctx, token)
}

// CheckTokenBlacklist 检查Token黑名单
func (s *AuthService) CheckTokenBlacklist(ctx context.Context, tokenID string) (bool, error) {
	return s.authUc.CheckTokenBlacklist(ctx, tokenID)
}

// GetUserSession 获取用户会话
func (s *AuthService) GetUserSession(ctx context.Context, userID int64) error {
	session, err := s.authUc.GetUserSession(ctx, userID)
	if err != nil {
		return err
	}

	if session.IsExpired() {
		return biz.ErrSessionExpired
	}

	return nil
}

// ValidateSession 验证会话有效性
func (s *AuthService) ValidateSession(ctx context.Context, userID int64, refreshToken string) (bool, error) {
	return s.authUc.ValidateSession(ctx, userID, refreshToken)
}

// RevokeAllUserTokens 撤销用户所有Token
func (s *AuthService) RevokeAllUserTokens(ctx context.Context, userID int64) error {
	s.log.WithContext(ctx).Infof("Revoke all tokens for user: %d", userID)

	err := s.authUc.RevokeAllUserTokens(ctx, userID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("revoke all user tokens failed: %v", err)
		return err
	}

	return nil
}

// GenerateTokenPair 生成Token对
func (s *AuthService) GenerateTokenPair(ctx context.Context, userID int64, username string) (*auth.TokenPair, error) {
	return s.jwtManager.GenerateTokenPair(userID, username)
}

// GetTokenExpiry 获取Token过期信息
func (s *AuthService) GetTokenExpiry(ctx context.Context, token string) (int64, error) {
	claims, err := s.jwtManager.VerifyToken(token)
	if err != nil {
		return 0, err
	}

	return claims.ExpiresAt.Unix(), nil
}
