package wpubsub

import (
	"context"
)

// Message represents a message to be published/consumed
type Message struct {
	Topic    string
	Key      string
	Value    []byte
	Metadata map[string]string
}

// Publisher defines the interface for publishing messages
type Publisher interface {
	Publish(ctx context.Context, message *Message) error
	Close() error
}

// Subscriber defines the interface for subscribing to messages
type Subscriber interface {
	Subscribe(ctx context.Context, topic string, handler func(context.Context, *Message) error) error
	Close() error
}

// PubSub combines both publisher and subscriber interfaces
type PubSub interface {
	Publisher
	Subscriber
}

// NotificationMessage represents a notification message for the event broker
type NotificationMessage struct {
	NotificationID string                 `json:"notification_id"`
	Title          string                 `json:"title"`
	Body           string                 `json:"body"`
	RefID          string                 `json:"ref_id"`
	Channels       []ChannelDestination   `json:"channels"`
	ScheduleAt     *int64                 `json:"schedule_at,omitempty"` // Unix timestamp
	Metadata       map[string]string      `json:"metadata,omitempty"`
}

// ChannelDestination represents a channel and its destination
type ChannelDestination struct {
	Channel     string `json:"channel"`
	Destination string `json:"destination"`
}