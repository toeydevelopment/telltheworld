package biz

import (
	"context"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/constant"
)

func TestNewPushProvider(t *testing.T) {
	logger := log.NewStdLogger(io.Discard)
	provider := NewPushProvider(logger)

	assert.NotNil(t, provider)
	assert.Implements(t, (*NotificationProvider)(nil), provider)
}

func TestPushProvider_GetChannel(t *testing.T) {
	logger := log.NewStdLogger(io.Discard)
	provider := NewPushProvider(logger)

	channel := provider.GetChannel()
	assert.Equal(t, constant.Push, channel)
}

func TestPushProvider_Send(t *testing.T) {
	tests := []struct {
		name        string
		destination string
		title       string
		body        string
		metadata    map[string]string
		wantErr     bool
	}{
		{
			name:        "success - basic push notification",
			destination: "device-token-123",
			title:       "New Message",
			body:        "You have a new message",
			metadata:    nil,
			wantErr:     false,
		},
		{
			name:        "success - push with metadata",
			destination: "fcm-token-abc",
			title:       "Alert",
			body:        "Important notification",
			metadata: map[string]string{
				"priority":  "high",
				"sound":     "default",
				"badge":     "1",
				"click_url": "myapp://notification/123",
			},
			wantErr: false,
		},
		{
			name:        "success - long device token",
			destination: "very-long-device-token-12345678901234567890123456789012345678901234567890",
			title:       "Test",
			body:        "Test body",
			metadata:    nil,
			wantErr:     false,
		},
		{
			name:        "success - empty body",
			destination: "token-456",
			title:       "Title Only",
			body:        "",
			metadata:    nil,
			wantErr:     false,
		},
		{
			name:        "success - unicode in title and body",
			destination: "token-789",
			title:       "สวัสดี Hello 你好",
			body:        "ข้อความทดสอบ Test message 测试消息",
			metadata:    nil,
			wantErr:     false,
		},
		{
			name:        "success - APNS token format",
			destination: "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
			title:       "iOS Push",
			body:        "Test iOS notification",
			metadata: map[string]string{
				"platform": "ios",
				"sound":    "default",
			},
			wantErr: false,
		},
		{
			name:        "success - FCM token format",
			destination: "fcm:APA91bHun4MxP5egoKMwt2KZFBaFUH",
			title:       "Android Push",
			body:        "Test Android notification",
			metadata: map[string]string{
				"platform": "android",
				"icon":     "notification_icon",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.NewStdLogger(io.Discard)
			provider := NewPushProvider(logger)

			err := provider.Send(context.Background(), tt.destination, tt.title, tt.body, tt.metadata)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPushProvider_Send_WithContext(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		wantErr bool
	}{
		{
			name:    "success - with background context",
			ctx:     context.Background(),
			wantErr: false,
		},
		{
			name:    "success - with todo context",
			ctx:     context.TODO(),
			wantErr: false,
		},
		{
			name: "success - with value context",
			ctx: context.WithValue(context.Background(), "request_id", "test-123"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.NewStdLogger(io.Discard)
			provider := NewPushProvider(logger)

			err := provider.Send(tt.ctx, "device-token-123", "Test", "Body", nil)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPushProvider_Send_Concurrent(t *testing.T) {
	logger := log.NewStdLogger(io.Discard)
	provider := NewPushProvider(logger)

	// Test concurrent sends
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			err := provider.Send(
				context.Background(),
				"device-token-123",
				"Concurrent Test",
				"Body",
				nil,
			)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestPushProvider_Comparison_With_EmailProvider(t *testing.T) {
	logger := log.NewStdLogger(io.Discard)

	pushProvider := NewPushProvider(logger)
	emailProvider := NewEmailProvider(logger)

	// Verify they are different providers
	assert.NotEqual(t, pushProvider.GetChannel(), emailProvider.GetChannel())
	assert.Equal(t, constant.Push, pushProvider.GetChannel())
	assert.Equal(t, constant.Email, emailProvider.GetChannel())
}
