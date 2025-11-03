package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/constant"
)

// PushProvider implements NotificationProvider for push notifications
type PushProvider struct {
	log *log.Helper
}

// NewPushProvider creates a new push notification provider
func NewPushProvider(logger log.Logger) NotificationProvider {
	return &PushProvider{
		log: log.NewHelper(log.With(logger, "module", "biz/push_provider")),
	}
}

// Send sends a push notification (dummy implementation)
func (p *PushProvider) Send(ctx context.Context, destination string, title string, body string, metadata map[string]string) error {
	p.log.WithContext(ctx).Infof("[PUSH] Sending notification to device token: %s", destination)
	p.log.WithContext(ctx).Infof("[PUSH] Title: %s", title)
	p.log.WithContext(ctx).Infof("[PUSH] Body: %s", body)

	// Simulate network delay
	time.Sleep(100 * time.Millisecond)

	// In a real implementation, this would connect to FCM, APNS, etc.
	// For now, just log the notification
	p.log.WithContext(ctx).Infof("[PUSH] ✓ Successfully sent push notification to %s", destination)

	return nil
}

// GetChannel returns the channel type this provider handles
func (p *PushProvider) GetChannel() constant.NotificationChannel {
	return constant.Push
}