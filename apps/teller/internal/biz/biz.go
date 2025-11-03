package biz

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	NewNotification,
	NewProviderRegistryWithProviders,
	NewNotificationWorker,
)

// NewProviderRegistryWithProviders creates and configures the provider registry
func NewProviderRegistryWithProviders(logger log.Logger) *ProviderRegistry {
	registry := NewProviderRegistry()

	// Register providers
	registry.Register(NewPushProvider(logger))
	registry.Register(NewEmailProvider(logger))

	return registry
}
