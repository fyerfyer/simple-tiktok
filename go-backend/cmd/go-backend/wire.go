//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"go-backend/internal/biz"
	"go-backend/internal/conf"
	"go-backend/internal/data"
	"go-backend/internal/middleware"
	"go-backend/internal/server"
	"go-backend/internal/service"
	"go-backend/pkg/auth"
	"go-backend/pkg/utils"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is providers.
var ProviderSet = wire.NewSet(
	server.ProviderSet,
	data.ProviderSet,
	biz.ProviderSet,
	service.ProviderSet,
	newJWTManager,
	newPasswordManager,
	newValidator,
	newAuthMiddleware,
)

// newJWTManager JWT manager provider
func newJWTManager(bc *conf.Bootstrap) *auth.JWTManager {
	return auth.NewJWTManager(
		bc.Jwt.Secret,
		bc.Jwt.ExpireTime.AsDuration(),
	)
}

// newPasswordManager password manager provider
func newPasswordManager() *auth.PasswordManager {
	return auth.NewPasswordManager()
}

// newValidator param validator provider
func newValidator() *utils.Validator {
	return utils.NewValidator()
}

// newAuthMiddleware auth middleware provider
func newAuthMiddleware(jwtManager *auth.JWTManager) *middleware.AuthMiddleware {
	return middleware.NewAuthMiddleware(jwtManager)
}

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, *conf.Bootstrap, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(ProviderSet, newApp))
}
