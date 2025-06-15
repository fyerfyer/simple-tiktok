package middleware

import (
	"github.com/google/wire"
)

// ProviderSet is middleware providers.
var ProviderSet = wire.NewSet(
	NewAuthMiddleware,
	NewRBACMiddleware,
	NewRateLimitMiddleware,
	NewSecurityMiddleware,
)
