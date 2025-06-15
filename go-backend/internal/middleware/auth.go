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

// JWT认证中间件
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

			// 验证Token
			claims, err := a.jwtManager.VerifyToken(token)
			if err != nil {
				a.log.WithContext(ctx).Warnf("invalid token: %v", err)
				return nil, NewAuthError(v1.ErrorCode_TOKEN_INVALID, "invalid token")
			}

			// 注入用户信息到上下文
			ctx = context.WithValue(ctx, "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "username", claims.Username)
			ctx = context.WithValue(ctx, "token_id", claims.TokenID)

			return handler(ctx, req)
		}
	}
}

// 可选JWT认证中间件
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
					// 注入用户信息到上下文
					ctx = context.WithValue(ctx, "user_id", claims.UserID)
					ctx = context.WithValue(ctx, "username", claims.Username)
					ctx = context.WithValue(ctx, "token_id", claims.TokenID)
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

			// 从请求中提取刷新Token
			refreshToken := a.extractRefreshToken(tr)
			if refreshToken == "" {
				return nil, NewAuthError(v1.ErrorCode_TOKEN_INVALID, "refresh token required")
			}

			// 验证刷新Token
			claims, err := a.jwtManager.VerifyRefreshToken(refreshToken)
			if err != nil {
				a.log.WithContext(ctx).Warnf("invalid refresh token: %v", err)
				return nil, NewAuthError(v1.ErrorCode_TOKEN_INVALID, "invalid refresh token")
			}

			// 注入用户信息到上下文
			ctx = context.WithValue(ctx, "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "username", claims.Username)
			ctx = context.WithValue(ctx, "refresh_token", refreshToken)
			ctx = context.WithValue(ctx, "token_id", claims.TokenID)

			return handler(ctx, req)
		}
	}
}

// 从请求中提取Token
func (a *AuthMiddleware) extractToken(tr transport.Transporter) string {
	// 从Header中获取
	if header := tr.RequestHeader(); header != nil {
		if auth := header.Get("Authorization"); auth != "" {
			if strings.HasPrefix(auth, "Bearer ") {
				return strings.TrimPrefix(auth, "Bearer ")
			}
		}

		// 直接从token header获取
		if token := header.Get("token"); token != "" {
			return token
		}
	}

	// 从Query参数中获取
	// 检查是否为 HTTP transport 并从Query参数中获取
	if ht, ok := tr.(http.Transporter); ok {
		req := ht.Request()
		if token := req.URL.Query().Get("token"); token != "" {
			return token
		}
	}

	return ""
}

// 从请求中提取Refresh Token
func (a *AuthMiddleware) extractRefreshToken(tr transport.Transporter) string {
	// 从Header中获取
	if header := tr.RequestHeader(); header != nil {
		if token := header.Get("refresh_token"); token != "" {
			return token
		}
	}

	// 从Query参数中获取
	if ht, ok := tr.(http.Transporter); ok {
		req := ht.Request()
		if token := req.URL.Query().Get("refresh_token"); token != "" {
			return token
		}
	}

	return ""
}

// 从上下文中获取用户ID
func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value("user_id").(int64)
	return userID, ok
}

// 从上下文中获取用户名
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value("username").(string)
	return username, ok
}

// 从上下文中获取TokenID
func GetTokenIDFromContext(ctx context.Context) (string, bool) {
	tokenID, ok := ctx.Value("token_id").(string)
	return tokenID, ok
}

// 从上下文中获取刷新Token
func GetRefreshTokenFromContext(ctx context.Context) (string, bool) {
	refreshToken, ok := ctx.Value("refresh_token").(string)
	return refreshToken, ok
}

// 创建认证错误
func NewAuthError(code v1.ErrorCode, message string) error {
	return errors.New(message)
}
