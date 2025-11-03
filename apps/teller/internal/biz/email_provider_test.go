package biz

import (
	"context"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/constant"
)

func TestNewEmailProvider(t *testing.T) {
	logger := log.NewStdLogger(io.Discard)
	provider := NewEmailProvider(logger)

	assert.NotNil(t, provider)
	assert.Implements(t, (*NotificationProvider)(nil), provider)
}

func TestEmailProvider_GetChannel(t *testing.T) {
	logger := log.NewStdLogger(io.Discard)
	provider := NewEmailProvider(logger)

	channel := provider.GetChannel()
	assert.Equal(t, constant.Email, channel)
}

func TestEmailProvider_Send(t *testing.T) {
	tests := []struct {
		name        string
		destination string
		title       string
		body        string
		metadata    map[string]string
		wantErr     bool
	}{
		{
			name:        "success - basic email",
			destination: "test@example.com",
			title:       "Test Subject",
			body:        "Test Body",
			metadata:    nil,
			wantErr:     false,
		},
		{
			name:        "success - email with metadata",
			destination: "user@example.com",
			title:       "Important Notice",
			body:        "This is an important message",
			metadata: map[string]string{
				"priority": "high",
				"template": "notification",
			},
			wantErr: false,
		},
		{
			name:        "success - long body",
			destination: "recipient@example.com",
			title:       "Newsletter",
			body:        "Lorem ipsum dolor sit amet, consectetur adipiscing elit. " +
						"Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
			metadata:    nil,
			wantErr:     false,
		},
		{
			name:        "success - empty body",
			destination: "test@example.com",
			title:       "Empty Body Test",
			body:        "",
			metadata:    nil,
			wantErr:     false,
		},
		{
			name:        "success - special characters in email",
			destination: "test+tag@example.com",
			title:       "Test",
			body:        "Test body",
			metadata:    nil,
			wantErr:     false,
		},
		{
			name:        "success - unicode in title and body",
			destination: "test@example.com",
			title:       "สวัสดี Hello 你好",
			body:        "ข้อความทดสอบ Test message 测试消息",
			metadata:    nil,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.NewStdLogger(io.Discard)
			provider := NewEmailProvider(logger)

			err := provider.Send(context.Background(), tt.destination, tt.title, tt.body, tt.metadata)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmailProvider_Send_WithContext(t *testing.T) {
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
			provider := NewEmailProvider(logger)

			err := provider.Send(tt.ctx, "test@example.com", "Test", "Body", nil)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmailProvider_Send_Concurrent(t *testing.T) {
	logger := log.NewStdLogger(io.Discard)
	provider := NewEmailProvider(logger)

	// Test concurrent sends
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(id int) {
			err := provider.Send(
				context.Background(),
				"test@example.com",
				"Concurrent Test",
				"Body",
				nil,
			)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}
}
