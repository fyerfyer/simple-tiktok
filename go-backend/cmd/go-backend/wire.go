//go:build wireinject
// +build wireinject

package main

import (
	"go-backend/internal/biz"
	"go-backend/internal/conf"
	"go-backend/internal/data"
	"go-backend/internal/data/producer"
	"go-backend/internal/middleware"
	"go-backend/internal/server"
	"go-backend/internal/service"
	"go-backend/pkg/auth"
	"go-backend/pkg/media"
	"go-backend/pkg/messaging"
	"go-backend/pkg/security"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, *conf.Business, *conf.Bootstrap, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(
		server.ProviderSet,
		data.ProviderSet,
		biz.ProviderSet,
		service.ProviderSet,
		middleware.ProviderSet,
		producer.ProviderSet,

		// pkg层的providers
		newJWTManager,
		newPasswordManager,
		newMemoryRBACManager,
		newSimplePermissionChecker,
		newValidator,
		newSessionManager,
		newKafkaManager,
		newVideoProcessor,

		// 接口绑定
		wire.Bind(new(biz.AuthRepo), new(*data.SessionRepo)),
		wire.Bind(new(biz.RoleRepo), new(*data.RoleRepo)),
		wire.Bind(new(biz.PermissionRepo), new(*data.PermissionRepo)),

		// 主应用构造器
		newApp,
	))
}

// Provider functions
func newJWTManager(bc *conf.Bootstrap) *auth.JWTManager {
	return auth.NewJWTManager(
		bc.Jwt.Secret,
		bc.Jwt.ExpireTime.AsDuration(),
	)
}

func newPasswordManager() *auth.PasswordManager {
	return auth.NewPasswordManager()
}

func newMemoryRBACManager() auth.RBACManager {
	return auth.NewMemoryRBACManager()
}

func newSimplePermissionChecker(rbacManager auth.RBACManager) auth.PermissionChecker {
	return auth.NewSimplePermissionChecker(rbacManager)
}

func newValidator() *security.Validator {
	return security.NewValidator()
}

func newSessionManager() auth.SessionManager {
	return auth.NewMemorySessionManager()
}

func newKafkaManager(dc *conf.Data, logger log.Logger) *messaging.KafkaManager {
	kafkaManager, _ := messaging.NewKafkaManager(dc.Kafka, logger)
	return kafkaManager
}

func newVideoProcessor(bc *conf.Business) *media.VideoProcessor {
	return media.NewVideoProcessor(
		bc.Video.MaxFileSize,
		bc.Video.SupportedFormats,
		int(bc.Video.CoverWidth),
		int(bc.Video.CoverHeight),
		int(bc.Video.CoverQuality),
	)
}
