package server

import (
	v1 "go-backend/api/user/v1"
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
	authMiddleware *middleware.AuthMiddleware,
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
	).Build()

	// 可选认证的路由中间件
	optionalAuth := selector.Server(
		authMiddleware.OptionalJWTAuth(),
	).Path(
		"/douyin/feed",
	).Build()

	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
			metrics.Server(),
			validate.Validator(),
			authRequired, // 在这里配置认证中间件
			optionalAuth, // 在这里配置可选认证中间件
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

	// 注册用户服务HTTP路由（只传两个参数）
	v1.RegisterUserServiceHTTPServer(srv, userService)

	return srv
}
