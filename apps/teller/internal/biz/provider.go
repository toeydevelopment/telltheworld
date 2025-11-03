package biz

import (
	"context"
	"fmt"

	"github.com/toeydevelopment/telltheworld/apps/teller/internal/constant"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/errors"
)

// NotificationProvider defines the interface for sending notifications
type NotificationProvider interface {
	// Send sends a notification to the specified destination
	Send(ctx context.Context, destination string, title string, body string, metadata map[string]string) error
	// GetChannel returns the channel type this provider handles
	GetChannel() constant.NotificationChannel
}

// ProviderRegistry manages notification providers
type ProviderRegistry struct {
	providers map[constant.NotificationChannel]NotificationProvider
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[constant.NotificationChannel]NotificationProvider),
	}
}

func (r *ProviderRegistry) Register(provider NotificationProvider) {
	r.providers[provider.GetChannel()] = provider
}

func (r *ProviderRegistry) GetProvider(channel constant.NotificationChannel) (NotificationProvider, error) {
	provider, exists := r.providers[channel]
	if !exists {
		return nil, fmt.Errorf("%w: %s", errors.ErrProviderNotFound, channel)
	}
	return provider, nil
}

func (r *ProviderRegistry) GetAllProviders() map[constant.NotificationChannel]NotificationProvider {
	return r.providers
}
