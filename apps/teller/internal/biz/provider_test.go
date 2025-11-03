package biz

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/constant"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/errors"
	"go.uber.org/mock/gomock"
)

func TestNewProviderRegistry(t *testing.T) {
	registry := NewProviderRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.providers)
	assert.Empty(t, registry.providers)
}

func TestProviderRegistry_Register(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*ProviderRegistry, *MockNotificationProvider)
		validate func(*testing.T, *ProviderRegistry)
	}{
		{
			name: "register single provider",
			setup: func(registry *ProviderRegistry, mockProvider *MockNotificationProvider) {
				mockProvider.EXPECT().
					GetChannel().
					Return(constant.Email)
				registry.Register(mockProvider)
			},
			validate: func(t *testing.T, registry *ProviderRegistry) {
				assert.Len(t, registry.providers, 1)
				_, exists := registry.providers[constant.Email]
				assert.True(t, exists)
			},
		},
		{
			name: "register multiple providers",
			setup: func(registry *ProviderRegistry, mockProvider *MockNotificationProvider) {
				// Register email provider
				mockProvider.EXPECT().
					GetChannel().
					Return(constant.Email)
				registry.Register(mockProvider)

				// Create another mock for push
				ctrl := gomock.NewController(t)
				pushProvider := NewMockNotificationProvider(ctrl)
				pushProvider.EXPECT().
					GetChannel().
					Return(constant.Push)
				registry.Register(pushProvider)
			},
			validate: func(t *testing.T, registry *ProviderRegistry) {
				assert.Len(t, registry.providers, 2)
				_, emailExists := registry.providers[constant.Email]
				_, pushExists := registry.providers[constant.Push]
				assert.True(t, emailExists)
				assert.True(t, pushExists)
			},
		},
		{
			name: "register same channel twice - overwrites",
			setup: func(registry *ProviderRegistry, mockProvider *MockNotificationProvider) {
				// Register first provider
				mockProvider.EXPECT().
					GetChannel().
					Return(constant.Email).
					Times(2)
				registry.Register(mockProvider)
				registry.Register(mockProvider)
			},
			validate: func(t *testing.T, registry *ProviderRegistry) {
				assert.Len(t, registry.providers, 1)
				_, exists := registry.providers[constant.Email]
				assert.True(t, exists)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			registry := NewProviderRegistry()
			mockProvider := NewMockNotificationProvider(ctrl)

			tt.setup(registry, mockProvider)
			tt.validate(t, registry)
		})
	}
}

func TestProviderRegistry_GetProvider(t *testing.T) {
	tests := []struct {
		name      string
		channel   constant.NotificationChannel
		setup     func(*ProviderRegistry, *gomock.Controller)
		wantErr   bool
		errMsg    string
		validate  func(*testing.T, NotificationProvider)
	}{
		{
			name:    "success - get registered provider",
			channel: constant.Email,
			setup: func(registry *ProviderRegistry, ctrl *gomock.Controller) {
				mockProvider := NewMockNotificationProvider(ctrl)
				mockProvider.EXPECT().
					GetChannel().
					Return(constant.Email)
				registry.Register(mockProvider)
			},
			wantErr: false,
			validate: func(t *testing.T, provider NotificationProvider) {
				assert.NotNil(t, provider)
			},
		},
		{
			name:    "failure - provider not registered",
			channel: constant.Push,
			setup: func(registry *ProviderRegistry, ctrl *gomock.Controller) {
				// Don't register any provider
			},
			wantErr: true,
			errMsg:  "no provider registered for channel",
		},
		{
			name:    "success - get one of multiple providers",
			channel: constant.Push,
			setup: func(registry *ProviderRegistry, ctrl *gomock.Controller) {
				// Register email provider
				emailProvider := NewMockNotificationProvider(ctrl)
				emailProvider.EXPECT().
					GetChannel().
					Return(constant.Email)
				registry.Register(emailProvider)

				// Register push provider
				pushProvider := NewMockNotificationProvider(ctrl)
				pushProvider.EXPECT().
					GetChannel().
					Return(constant.Push)
				registry.Register(pushProvider)
			},
			wantErr: false,
			validate: func(t *testing.T, provider NotificationProvider) {
				assert.NotNil(t, provider)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			registry := NewProviderRegistry()
			tt.setup(registry, ctrl)

			provider, err := registry.GetProvider(tt.channel)

			if tt.wantErr {
				assert.Error(t, err)
				assert.True(t, stderrors.Is(err, errors.ErrProviderNotFound), "expected ErrProviderNotFound")
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, provider)
				}
			}
		})
	}
}

func TestProviderRegistry_GetAllProviders(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*ProviderRegistry, *gomock.Controller)
		validate func(*testing.T, map[constant.NotificationChannel]NotificationProvider)
	}{
		{
			name: "empty registry",
			setup: func(registry *ProviderRegistry, ctrl *gomock.Controller) {
				// Don't register anything
			},
			validate: func(t *testing.T, providers map[constant.NotificationChannel]NotificationProvider) {
				assert.Empty(t, providers)
			},
		},
		{
			name: "single provider",
			setup: func(registry *ProviderRegistry, ctrl *gomock.Controller) {
				mockProvider := NewMockNotificationProvider(ctrl)
				mockProvider.EXPECT().
					GetChannel().
					Return(constant.Email)
				registry.Register(mockProvider)
			},
			validate: func(t *testing.T, providers map[constant.NotificationChannel]NotificationProvider) {
				assert.Len(t, providers, 1)
				_, exists := providers[constant.Email]
				assert.True(t, exists)
			},
		},
		{
			name: "multiple providers",
			setup: func(registry *ProviderRegistry, ctrl *gomock.Controller) {
				emailProvider := NewMockNotificationProvider(ctrl)
				emailProvider.EXPECT().
					GetChannel().
					Return(constant.Email)
				registry.Register(emailProvider)

				pushProvider := NewMockNotificationProvider(ctrl)
				pushProvider.EXPECT().
					GetChannel().
					Return(constant.Push)
				registry.Register(pushProvider)
			},
			validate: func(t *testing.T, providers map[constant.NotificationChannel]NotificationProvider) {
				assert.Len(t, providers, 2)
				_, emailExists := providers[constant.Email]
				_, pushExists := providers[constant.Push]
				assert.True(t, emailExists)
				assert.True(t, pushExists)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			registry := NewProviderRegistry()
			tt.setup(registry, ctrl)

			providers := registry.GetAllProviders()
			tt.validate(t, providers)
		})
	}
}

func TestNotificationProvider_Interface(t *testing.T) {
	// This test ensures that the mock implementation satisfies the interface
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := NewMockNotificationProvider(ctrl)

	// Set up expectations
	mockProvider.EXPECT().
		GetChannel().
		Return(constant.Email)
	mockProvider.EXPECT().
		Send(gomock.Any(), "test@example.com", "Title", "Body", gomock.Any()).
		Return(nil)

	// Test that the mock can be used as the interface
	var provider NotificationProvider = mockProvider

	// Test GetChannel
	channel := provider.GetChannel()
	assert.Equal(t, constant.Email, channel)

	// Test Send
	err := provider.Send(context.Background(), "test@example.com", "Title", "Body", nil)
	assert.NoError(t, err)
}
