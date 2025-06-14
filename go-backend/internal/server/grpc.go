// filepath: go-backend/internal/server/grpc.go
package server

import (
	"context"

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
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(
	c *conf.Server,
	userService *service.UserService,
	authMiddleware *middleware.AuthMiddleware,
	logger log.Logger,
) *grpc.Server {
	// 需要认证的gRPC方法选择器 - 使用正确的函数签名
	authRequired := selector.Server(
		authMiddleware.JWTAuth(),
	).Match(func(ctx context.Context, operation string) bool {
		// gRPC内部调用接口不需要JWT认证
		internalMethods := []string{
			"/user.v1.UserService/GetUserInfo",
			"/user.v1.UserService/GetUsersInfo",
			"/user.v1.UserService/VerifyToken",
			"/user.v1.UserService/UpdateUserStats",
		}

		for _, method := range internalMethods {
			if operation == method {
				return false
			}
		}

		// 公开接口不需要认证
		publicMethods := []string{
			"/user.v1.UserService/Register",
			"/user.v1.UserService/Login",
		}

		for _, method := range publicMethods {
			if operation == method {
				return false
			}
		}

		return true
	}).Build()

	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
			metrics.Server(),
			validate.Validator(),
			authRequired, // 在这里配置认证中间件
		),
	}

	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}

	srv := grpc.NewServer(opts...)

	// 注册用户服务gRPC（只传两个参数）
	v1.RegisterUserServiceServer(srv, userService)

	return srv
}
