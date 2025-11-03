package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/constant"
)

// EmailProvider implements NotificationProvider for email notifications
type EmailProvider struct {
	log *log.Helper
}

// NewEmailProvider creates a new email notification provider
func NewEmailProvider(logger log.Logger) NotificationProvider {
	return &EmailProvider{
		log: log.NewHelper(log.With(logger, "module", "biz/email_provider")),
	}
}

// Send sends an email notification (dummy implementation)
func (e *EmailProvider) Send(ctx context.Context, destination string, title string, body string, metadata map[string]string) error {
	e.log.WithContext(ctx).Infof("[EMAIL] Sending notification to email: %s", destination)
	e.log.WithContext(ctx).Infof("[EMAIL] Subject: %s", title)
	e.log.WithContext(ctx).Infof("[EMAIL] Body: %s", body)

	// Simulate network delay
	time.Sleep(200 * time.Millisecond)

	// In a real implementation, this would connect to SMTP server, SendGrid, SES, etc.
	// For now, just log the notification
	e.log.WithContext(ctx).Infof("[EMAIL] ✓ Successfully sent email to %s", destination)

	return nil
}

// GetChannel returns the channel type this provider handles
func (e *EmailProvider) GetChannel() constant.NotificationChannel {
	return constant.Email
}