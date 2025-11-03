package wpubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/nats-io/nats.go"
)

type NATSPubSub struct {
	conn       *nats.Conn
	js         nats.JetStreamContext
	streamName string
	log        *log.Helper
}

// NewNATSPubSub creates a new NATS pubsub instance with JetStream
func NewNATSPubSub(natsURL string, streamName string, logger log.Logger) (*NATSPubSub, error) {
	// Connect to NATS
	conn, err := nats.Connect(natsURL,
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(-1), // Infinite reconnects
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				log.Errorf("Disconnected from NATS: %v", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Infof("Reconnected to NATS at %s", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Errorf("NATS connection closed")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Create or update stream
	streamConfig := &nats.StreamConfig{
		Name:      streamName,
		Subjects:  []string{streamName + ".>"},
		Retention: nats.WorkQueuePolicy, // Messages are removed after being acknowledged
		Storage:   nats.FileStorage,
		MaxAge:    7 * 24 * time.Hour, // Messages older than 7 days are removed
		Discard:   nats.DiscardOld,
	}

	// Check if stream exists and create/update it
	stream, err := js.StreamInfo(streamName)
	if err != nil {
		// Stream doesn't exist, create it
		_, err = js.AddStream(streamConfig)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create stream: %w", err)
		}
		log.Infof("Created NATS JetStream: %s", streamName)
	} else {
		// Stream exists, update it if needed
		_, err = js.UpdateStream(streamConfig)
		if err != nil {
			log.Warnf("Failed to update stream %s: %v", streamName, err)
		} else {
			log.Infof("Updated NATS JetStream: %s", streamName)
		}
		log.Infof("Using existing NATS JetStream: %s (messages: %d)", streamName, stream.State.Msgs)
	}

	return &NATSPubSub{
		conn:       conn,
		js:         js,
		streamName: streamName,
		log:        log.NewHelper(log.With(logger, "module", "wpubsub/nats")),
	}, nil
}

// Publish publishes a message to NATS JetStream
func (n *NATSPubSub) Publish(ctx context.Context, message *Message) error {
	// Serialize the message value
	data := message.Value

	// Create NATS message headers
	headers := nats.Header{}
	headers.Set("Topic", message.Topic)
	if message.Key != "" {
		headers.Set("Key", message.Key)
	}
	for k, v := range message.Metadata {
		headers.Set(k, v)
	}

	// Construct the subject as streamName.topic
	subject := fmt.Sprintf("%s.%s", n.streamName, message.Topic)

	// Publish to JetStream with headers
	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  headers,
	}

	// Publish with acknowledgment
	pubAck, err := n.js.PublishMsg(msg, nats.Context(ctx))
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	n.log.Debugf("Published message to %s (sequence: %d, stream: %s)", subject, pubAck.Sequence, pubAck.Stream)
	return nil
}

// Subscribe subscribes to messages from NATS JetStream
func (n *NATSPubSub) Subscribe(ctx context.Context, topic string, handler func(context.Context, *Message) error) error {
	// Construct the subject
	subject := fmt.Sprintf("%s.%s", n.streamName, topic)

	// Create a durable consumer with explicit acknowledgment
	// Replace dots and other special characters with underscores for valid consumer name
	sanitizedTopic := strings.ReplaceAll(topic, ".", "_")
	sanitizedTopic = strings.ReplaceAll(sanitizedTopic, "-", "_")
	consumerName := fmt.Sprintf("%s_%s_consumer", n.streamName, sanitizedTopic)

	// Configure the consumer
	consumerConfig := &nats.ConsumerConfig{
		Durable:         consumerName,
		AckPolicy:       nats.AckExplicitPolicy,
		DeliverPolicy:   nats.DeliverAllPolicy,
		ReplayPolicy:    nats.ReplayInstantPolicy,
		MaxDeliver:      5, // Max retry attempts
		AckWait:         30 * time.Second,
		FilterSubject:   subject,
		MaxAckPending:   100,
	}

	// Create or update the consumer
	_, err := n.js.AddConsumer(n.streamName, consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	// Create subscription
	sub, err := n.js.PullSubscribe(subject, consumerName)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	n.log.Infof("Subscribed to topic: %s (consumer: %s)", topic, consumerName)

	// Start message processing in a goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				n.log.Info("Subscription context cancelled, stopping consumer")
				sub.Unsubscribe()
				return
			default:
				// Fetch messages with a timeout
				msgs, err := sub.Fetch(10, nats.MaxWait(5*time.Second))
				if err != nil {
					if err != nats.ErrTimeout {
						n.log.Errorf("Error fetching messages: %v", err)
						time.Sleep(1 * time.Second)
					}
					continue
				}

				// Process messages
				for _, natsMsg := range msgs {
					// Convert NATS message to our Message type
					msg := &Message{
						Topic:    topic,
						Key:      natsMsg.Header.Get("Key"),
						Value:    natsMsg.Data,
						Metadata: make(map[string]string),
					}

					// Copy headers to metadata
					for k := range natsMsg.Header {
						if k != "Key" && k != "Topic" {
							msg.Metadata[k] = natsMsg.Header.Get(k)
						}
					}

					// Handle the message
					err := handler(ctx, msg)
					if err != nil {
						n.log.Errorf("Error handling message: %v", err)
						// Negative acknowledgment for retry
						if nakErr := natsMsg.Nak(); nakErr != nil {
							n.log.Errorf("Failed to NAK message: %v", nakErr)
						}
					} else {
						// Acknowledge successful processing
						if ackErr := natsMsg.Ack(); ackErr != nil {
							n.log.Errorf("Failed to ACK message: %v", ackErr)
						}
					}
				}
			}
		}
	}()

	return nil
}

// Close closes the NATS connection
func (n *NATSPubSub) Close() error {
	n.log.Info("Closing NATS connection")
	n.conn.Close()
	return nil
}

// PublishNotification is a helper method to publish notification messages
func (n *NATSPubSub) PublishNotification(ctx context.Context, notification *NotificationMessage) error {
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
			"timestamp":       fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	return n.Publish(ctx, msg)
}