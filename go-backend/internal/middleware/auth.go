package middleware

import (
	"context"
	"errors"
	"strings"

	"go-backend/pkg/auth"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
)

type AuthMiddleware struct {
	jwtManager *auth.JWTManager
}

func NewAuthMiddleware(jwtManager *auth.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
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
				return nil, errors.New("token required")
			}

			claims, err := a.jwtManager.VerifyToken(token)
			if err != nil {
				return nil, errors.New("invalid token")
			}

			ctx = context.WithValue(ctx, "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "username", claims.Username)

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
					ctx = context.WithValue(ctx, "user_id", claims.UserID)
					ctx = context.WithValue(ctx, "username", claims.Username)
				}
			}

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
