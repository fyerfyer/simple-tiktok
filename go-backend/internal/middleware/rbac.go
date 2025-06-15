package middleware

import (
	"context"
	"strings"

	"go-backend/api/common/v1"
	"go-backend/pkg/auth"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// RBACMiddleware RBAC权限中间件
type RBACMiddleware struct {
	permissionChecker auth.PermissionChecker
	log               *log.Helper
}

// NewRBACMiddleware 创建RBAC中间件
func NewRBACMiddleware(permissionChecker auth.PermissionChecker, logger log.Logger) *RBACMiddleware {
	return &RBACMiddleware{
		permissionChecker: permissionChecker,
		log:               log.NewHelper(logger),
	}
}

// ResourceAction 资源权限检查中间件
func (m *RBACMiddleware) ResourceAction() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 获取用户ID
			userID, ok := GetUserIDFromContext(ctx)
			if !ok {
				return nil, NewAuthError(v1.ErrorCode_TOKEN_INVALID, "token required")
			}

			// 获取资源和操作
			resource, action := m.extractResourceAction(ctx)
			if resource == "" || action == "" {
				// 无需权限检查的接口直接通过
				return handler(ctx, req)
			}

			// 检查权限
			hasPermission, err := m.permissionChecker.CheckPermission(ctx, userID, resource, action)
			if err != nil {
				m.log.WithContext(ctx).Errorf("check permission failed: %v", err)
				return nil, NewAuthError(v1.ErrorCode_SERVER_ERROR, "permission check failed")
			}

			if !hasPermission {
				m.log.WithContext(ctx).Warnf("permission denied: user=%d, resource=%s, action=%s", userID, resource, action)
				return nil, NewAuthError(v1.ErrorCode_PERMISSION_DENIED, "permission denied")
			}

			return handler(ctx, req)
		}
	}
}

// AdminOnly 管理员权限检查中间件
func (m *RBACMiddleware) AdminOnly() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			userID, ok := GetUserIDFromContext(ctx)
			if !ok {
				return nil, NewAuthError(v1.ErrorCode_TOKEN_INVALID, "token required")
			}

			isAdmin, err := m.permissionChecker.IsAdmin(ctx, userID)
			if err != nil {
				m.log.WithContext(ctx).Errorf("check admin failed: %v", err)
				return nil, NewAuthError(v1.ErrorCode_SERVER_ERROR, "admin check failed")
			}

			if !isAdmin {
				m.log.WithContext(ctx).Warnf("admin permission denied: user=%d", userID)
				return nil, NewAuthError(v1.ErrorCode_PERMISSION_DENIED, "admin permission required")
			}

			return handler(ctx, req)
		}
	}
}

// ModeratorOrAdmin 审核员或管理员权限检查
func (m *RBACMiddleware) ModeratorOrAdmin() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			userID, ok := GetUserIDFromContext(ctx)
			if !ok {
				return nil, NewAuthError(v1.ErrorCode_TOKEN_INVALID, "token required")
			}

			canModerate, err := m.permissionChecker.CanModerateContent(ctx, userID)
			if err != nil {
				m.log.WithContext(ctx).Errorf("check moderator failed: %v", err)
				return nil, NewAuthError(v1.ErrorCode_SERVER_ERROR, "permission check failed")
			}

			if !canModerate {
				m.log.WithContext(ctx).Warnf("moderator permission denied: user=%d", userID)
				return nil, NewAuthError(v1.ErrorCode_PERMISSION_DENIED, "moderator permission required")
			}

			return handler(ctx, req)
		}
	}
}

// extractResourceAction 从请求中提取资源和操作
func (m *RBACMiddleware) extractResourceAction(ctx context.Context) (string, string) {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return "", ""
	}

	// 获取操作路径
	var path, method string
	if ht, ok := tr.(http.Transporter); ok {
		req := ht.Request()
		path = req.URL.Path
		method = req.Method
	}

	// 资源和操作映射
	resourceMap := map[string]string{
		"/douyin/user":                   "/user",
		"/douyin/video/publish":          "/video",
		"/douyin/video/list":             "/video",
		"/douyin/video/feed":             "/video",
		"/douyin/favorite/action":        "/favorite",
		"/douyin/favorite/list":          "/favorite",
		"/douyin/comment/action":         "/comment",
		"/douyin/comment/list":           "/comment",
		"/douyin/relation/action":        "/relation",
		"/douyin/relation/follow/list":   "/relation",
		"/douyin/relation/follower/list": "/relation",
		"/douyin/relation/friend/list":   "/relation",
		"/douyin/message/action":         "/message",
		"/douyin/message/chat":           "/message",
	}

	// 管理员接口
	if strings.HasPrefix(path, "/douyin/admin/") {
		return "/*", "*"
	}

	// 获取资源
	resource := resourceMap[path]
	if resource == "" {
		// 动态匹配
		if strings.Contains(path, "/video/") {
			resource = "/video"
		} else if strings.Contains(path, "/user/") {
			resource = "/user"
		} else if strings.Contains(path, "/comment/") {
			resource = "/comment"
		}
	}

	// 操作类型映射
	action := strings.ToUpper(method)

	// 特殊操作处理
	if path == "/douyin/video/publish" && method == "POST" {
		action = "POST"
	} else if strings.Contains(path, "/action") && method == "POST" {
		action = "POST"
	} else if strings.Contains(path, "/list") && method == "GET" {
		action = "GET"
	}

	return resource, action
}

// CheckVideoPermission 检查视频权限
func (m *RBACMiddleware) CheckVideoPermission(action string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			userID, ok := GetUserIDFromContext(ctx)
			if !ok {
				return nil, NewAuthError(v1.ErrorCode_TOKEN_INVALID, "token required")
			}

			hasPermission, err := m.permissionChecker.CheckPermission(ctx, userID, "/video", action)
			if err != nil {
				return nil, err
			}

			if !hasPermission {
				return nil, NewAuthError(v1.ErrorCode_PERMISSION_DENIED, "video permission denied")
			}

			return handler(ctx, req)
		}
	}
}

// CheckCommentPermission 检查评论权限
func (m *RBACMiddleware) CheckCommentPermission(action string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			userID, ok := GetUserIDFromContext(ctx)
			if !ok {
				return nil, NewAuthError(v1.ErrorCode_TOKEN_INVALID, "token required")
			}

			hasPermission, err := m.permissionChecker.CheckPermission(ctx, userID, "/comment", action)
			if err != nil {
				return nil, err
			}

			if !hasPermission {
				return nil, NewAuthError(v1.ErrorCode_PERMISSION_DENIED, "comment permission denied")
			}

			return handler(ctx, req)
		}
	}
}

// SelfOrAdmin 检查是否为本人或管理员
func (m *RBACMiddleware) SelfOrAdmin(targetUserIDFunc func(interface{}) int64) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			currentUserID, ok := GetUserIDFromContext(ctx)
			if !ok {
				return nil, NewAuthError(v1.ErrorCode_TOKEN_INVALID, "token required")
			}

			targetUserID := targetUserIDFunc(req)

			// 检查是否为本人
			if currentUserID == targetUserID {
				return handler(ctx, req)
			}

			// 检查是否为管理员
			isAdmin, err := m.permissionChecker.IsAdmin(ctx, currentUserID)
			if err != nil {
				return nil, err
			}

			if !isAdmin {
				return nil, NewAuthError(v1.ErrorCode_PERMISSION_DENIED, "permission denied")
			}

			return handler(ctx, req)
		}
	}
}
