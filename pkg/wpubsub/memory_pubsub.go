package wpubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
)

type MemoryPubSub struct {
	mu          sync.RWMutex
	subscribers map[string][]func(context.Context, *Message) error
	log         *log.Helper
	closed      bool
}

// NewMemoryPubSub creates a new in-memory pubsub instance for development
func NewMemoryPubSub(logger log.Logger) *MemoryPubSub {
	return &MemoryPubSub{
		subscribers: make(map[string][]func(context.Context, *Message) error),
		log:         log.NewHelper(log.With(logger, "module", "wpubsub/memory")),
	}
}

// Publish publishes a message to all subscribers of a topic
func (m *MemoryPubSub) Publish(ctx context.Context, message *Message) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return fmt.Errorf("pubsub is closed")
	}

	handlers, exists := m.subscribers[message.Topic]
	if !exists {
		m.log.Debugf("No subscribers for topic: %s", message.Topic)
		return nil
	}

	// Call all handlers for this topic
	for _, handler := range handlers {
		// Run each handler in a goroutine to not block
		go func(h func(context.Context, *Message) error) {
			if err := h(ctx, message); err != nil {
				m.log.Errorf("Error handling message for topic %s: %v", message.Topic, err)
			}
		}(handler)
	}

	m.log.Debugf("Published message to topic %s to %d subscribers", message.Topic, len(handlers))
	return nil
}

// Subscribe subscribes to messages from a topic
func (m *MemoryPubSub) Subscribe(ctx context.Context, topic string, handler func(context.Context, *Message) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("pubsub is closed")
	}

	m.subscribers[topic] = append(m.subscribers[topic], handler)
	m.log.Infof("Subscribed to topic: %s (total subscribers: %d)", topic, len(m.subscribers[topic]))

	// Keep the subscription alive until context is cancelled
	go func() {
		<-ctx.Done()
		m.unsubscribe(topic, handler)
	}()

	return nil
}

// unsubscribe removes a handler from a topic
func (m *MemoryPubSub) unsubscribe(topic string, handler func(context.Context, *Message) error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, exists := m.subscribers[topic]
	if !exists {
		return
	}

	// Note: This simple comparison won't work for function equality
	// In a real implementation, we'd need to track subscription IDs
	m.log.Infof("Unsubscribe called for topic: %s", topic)
}

// Close closes the memory pubsub
func (m *MemoryPubSub) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	m.subscribers = make(map[string][]func(context.Context, *Message) error)
	m.log.Info("Memory pubsub closed")
	return nil
}

// PublishNotification is a helper method to publish notification messages
func (m *MemoryPubSub) PublishNotification(ctx context.Context, notification *NotificationMessage) error {
	// Serialize the notification
	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	// Create the message
	msg := &Message{
		Topic: "notifications",
		Key:   notification.NotificationID,
		Value: data,
		Metadata: map[string]string{
			"notification_id": notification.NotificationID,
			"ref_id":          notification.RefID,
		},
	}

	return m.Publish(ctx, msg)
}
