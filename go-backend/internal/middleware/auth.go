package middleware

import (
	"context"
	"errors"
	"strings"

	"go-backend/api/common/v1"
	"go-backend/pkg/auth"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
)

type AuthMiddleware struct {
	jwtManager *auth.JWTManager
	log        *log.Helper
}

func NewAuthMiddleware(jwtManager *auth.JWTManager, logger log.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
		log:        log.NewHelper(logger),
	}
}

// JWTAuth JWT认证中间件
func (a *AuthMiddleware) JWTAuth() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, errors.New("transport not found")
			}

			token := a.extractToken(tr)
			if token == "" {
				return nil, NewAuthError(v1.ErrorCode_TOKEN_INVALID, "token required")
			}

			claims, err := a.jwtManager.VerifyToken(token)
			if err != nil {
				a.log.WithContext(ctx).Warnf("invalid token: %v", err)
				return nil, NewAuthError(v1.ErrorCode_TOKEN_INVALID, "invalid token")
			}

			ctx = WithUserID(ctx, claims.UserID)
			ctx = WithUsername(ctx, claims.Username)
			ctx = WithTokenID(ctx, claims.TokenID)

			return handler(ctx, req)
		}
	}
}

// OptionalJWTAuth 可选JWT认证中间件
func (a *AuthMiddleware) OptionalJWTAuth() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			token := a.extractToken(tr)
			if token != "" {
				claims, err := a.jwtManager.VerifyToken(token)
				if err == nil {
					ctx = WithUserID(ctx, claims.UserID)
					ctx = WithUsername(ctx, claims.Username)
					ctx = WithTokenID(ctx, claims.TokenID)
				}
			}

			return handler(ctx, req)
		}
	}
}

// RefreshTokenAuth 刷新Token中间件
func (a *AuthMiddleware) RefreshTokenAuth() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, errors.New("transport not found")
			}

			refreshToken := a.extractRefreshToken(tr)
			if refreshToken == "" {
				return nil, NewAuthError(v1.ErrorCode_TOKEN_INVALID, "refresh token required")
			}

			claims, err := a.jwtManager.VerifyRefreshToken(refreshToken)
			if err != nil {
				a.log.WithContext(ctx).Warnf("invalid refresh token: %v", err)
				return nil, NewAuthError(v1.ErrorCode_TOKEN_INVALID, "invalid refresh token")
			}

			ctx = WithUserID(ctx, claims.UserID)
			ctx = WithUsername(ctx, claims.Username)
			ctx = WithRefreshToken(ctx, refreshToken)
			ctx = WithTokenID(ctx, claims.TokenID)

			return handler(ctx, req)
		}
	}
}

// extractToken 从请求中提取Token
func (a *AuthMiddleware) extractToken(tr transport.Transporter) string {
	if header := tr.RequestHeader(); header != nil {
		if auth := header.Get("Authorization"); auth != "" {
			if strings.HasPrefix(auth, "Bearer ") {
				return strings.TrimPrefix(auth, "Bearer ")
			}
		}

		if token := header.Get("token"); token != "" {
			return token
		}
	}

	if ht, ok := tr.(http.Transporter); ok {
		req := ht.Request()
		if token := req.URL.Query().Get("token"); token != "" {
			return token
		}
	}

	return ""
}

// extractRefreshToken 从请求中提取Refresh Token
func (a *AuthMiddleware) extractRefreshToken(tr transport.Transporter) string {
	if header := tr.RequestHeader(); header != nil {
		if token := header.Get("refresh_token"); token != "" {
			return token
		}
	}

	if ht, ok := tr.(http.Transporter); ok {
		req := ht.Request()
		if token := req.URL.Query().Get("refresh_token"); token != "" {
			return token
		}
	}

	return ""
}

// GetUserIDFromToken 从Token解析用户ID
func GetUserIDFromToken(ctx context.Context, token string) (int64, bool) {
	// 这是一个简化实现，实际项目中应该注入JWTManager
	if token == "" {
		return 0, false
	}

	// 临时实现：从上下文获取
	userID, ok := GetUserIDFromContext(ctx)
	return userID, ok
}

// NewAuthError 创建认证错误
func NewAuthError(code v1.ErrorCode, message string) error {
	return errors.New(message)
}
