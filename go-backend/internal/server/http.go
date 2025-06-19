package server

import (
	userv1 "go-backend/api/user/v1"
	videov1 "go-backend/api/video/v1"
	"go-backend/internal/conf"
	"go-backend/internal/middleware"
	"go-backend/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(
	c *conf.Server,
	userService *service.UserService,
	videoService *service.VideoService,
	authMiddleware *middleware.AuthMiddleware,
	rbacMiddleware *middleware.RBACMiddleware,
	rateLimitMiddleware *middleware.RateLimitMiddleware,
	securityMiddleware *middleware.SecurityMiddleware,
	videoMiddleware *middleware.VideoMiddleware,
	logger log.Logger,
) *http.Server {
	// 需要认证的路由中间件
	authRequired := selector.Server(
		authMiddleware.JWTAuth(),
	).Path(
		"/douyin/user",
		"/douyin/relation/action",
		"/douyin/relation/follow/list",
		"/douyin/relation/follower/list",
		"/douyin/relation/friend/list",
		"/douyin/publish/action",
		"/douyin/publish/list",
	).Build()

	// 可选认证的路由中间件
	optionalAuth := selector.Server(
		authMiddleware.OptionalJWTAuth(),
	).Path(
		"/douyin/feed",
	).Build()

	// 需要权限检查的路由中间件
	permissionRequired := selector.Server(
		rbacMiddleware.ResourceAction(),
	).Path(
		"/douyin/video/delete",   // 需要特定权限
		"/douyin/comment/delete", // 需要特定权限
		"/douyin/admin",          // 需要管理员权限
	).Build()

	// 限流中间件
	rateLimiter := rateLimitMiddleware.Limit()

	// 安全中间件
	security := securityMiddleware.GlobalSecurityHandler()

	// 视频中间件
	videoFileUploadValidator := videoMiddleware.FileUploadValidator()
	videoFileSizelimitor := videoMiddleware.FileSizeLimit()
	videoTitleValidator := videoMiddleware.VideoTitleValidator()
	videoFormatValidator := videoMiddleware.VideoFormatValidator()

	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),      // 恢复中间件
			logging.Server(logger),   // 日志中间件
			metrics.Server(),         // 指标中间件
			validate.Validator(),     // 验证器中间件
			security,                 // 全局安全中间件
			rateLimiter,              // 限流中间件
			authRequired,             // 认证中间件
			optionalAuth,             // 可选认证中间件
			permissionRequired,       // 权限中间件
			videoFileUploadValidator, // 视频文件上传验证中间件
			videoFileSizelimitor,     // 视频文件大小限制中间件
			videoTitleValidator,      // 视频标题验证中间件
			videoFormatValidator,     // 视频文件类型验证中间件
		),
	}

	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}

	srv := http.NewServer(opts...)

	// 注册用户服务HTTP路由
	userv1.RegisterUserServiceHTTPServer(srv, userService)

	// 注册视频服务HTTP路由
	videov1.RegisterVideoServiceHTTPServer(srv, videoService)

	return srv
}
