package domain

import (
	"github.com/google/wire"
)

// ProviderSet is domain providers.
var ProviderSet = wire.NewSet(
	NewEventFactory,
)
