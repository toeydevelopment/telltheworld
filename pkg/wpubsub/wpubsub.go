package wpubsub

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
)

// WPubSub implements PubSub interface
// This is a configurable implementation that can use different backends
type WPubSub struct {
	pubsubType string
	log        *log.Helper
	// For now, using in-memory implementation
	subscribers map[string][]func(context.Context, *Message) error
	mu          sync.RWMutex
}

// NewWPubSub creates a new pubsub implementation
func NewWPubSub(pubsubType string, logger log.Logger) (PubSub, error) {
	logHelper := log.NewHelper(log.With(logger, "module", "wpubsub"))

	// For now, we'll use a simple in-memory implementation
	// This can be replaced with actual broker integrations later
	switch pubsubType {
	case "kafka":
		logHelper.Info("Kafka pubsub configured (using in-memory for now)")
	case "redis":
		logHelper.Info("Redis pubsub configured (using in-memory for now)")
	case "memory":
		logHelper.Info("Using in-memory pubsub")
	default:
		logHelper.Warn("No pubsub type specified, using in-memory pubsub")
	}

	return &WPubSub{
		pubsubType:  pubsubType,
		log:         logHelper,
		subscribers: make(map[string][]func(context.Context, *Message) error),
	}, nil
}

// Publish publishes a message to the specified topic
func (w *WPubSub) Publish(ctx context.Context, message *Message) error {
	w.log.WithContext(ctx).Debugf("Publishing message to topic: %s", message.Topic)

	// In-memory implementation
	w.mu.RLock()
	handlers := w.subscribers[message.Topic]
	w.mu.RUnlock()

	for _, handler := range handlers {
		// Run handlers asynchronously
		go func(h func(context.Context, *Message) error) {
			if err := h(ctx, message); err != nil {
				w.log.WithContext(ctx).Errorf("Handler error: %v", err)
			}
		}(handler)
	}

	return nil
}

// Subscribe subscribes to a topic and processes messages with the handler
func (w *WPubSub) Subscribe(ctx context.Context, topic string, handler func(context.Context, *Message) error) error {
	w.log.WithContext(ctx).Infof("Subscribing to topic: %s", topic)

	w.mu.Lock()
	w.subscribers[topic] = append(w.subscribers[topic], handler)
	w.mu.Unlock()

	// In a real implementation, this would maintain the subscription
	// and handle incoming messages continuously
	return nil
}

// Close closes the pubsub connection
func (w *WPubSub) Close() error {
	w.log.Info("Closing pubsub connection")
	w.mu.Lock()
	w.subscribers = make(map[string][]func(context.Context, *Message) error)
	w.mu.Unlock()
	return nil
}

// PublishNotification is a helper function to publish notification messages
func PublishNotification(ctx context.Context, publisher Publisher, notification *NotificationMessage) error {
	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	message := &Message{
		Topic: "notifications.pending", // Default topic for pending notifications
		Key:   notification.NotificationID,
		Value: data,
		Metadata: map[string]string{
			"notification_id": notification.NotificationID,
			"ref_id":         notification.RefID,
		},
	}

	return publisher.Publish(ctx, message)
}