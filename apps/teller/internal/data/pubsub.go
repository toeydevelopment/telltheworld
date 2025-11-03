package data

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/conf"
	"github.com/toeydevelopment/telltheworld/pkg/wpubsub"
)

// NewPubSub creates a new pubsub instance based on configuration
func NewPubSub(cfg *conf.PubSubConfig, logger log.Logger) (wpubsub.PubSub, func(), error) {
	log := log.NewHelper(log.With(logger, "module", "data/pubsub"))

	var ps wpubsub.PubSub
	var cleanup func()

	switch cfg.Type {
	case "nats":
		natsURL := cfg.NatsUrl
		if natsURL == "" {
			natsURL = "nats://localhost:4222"
		}

		streamName := cfg.Config["stream_name"]
		if streamName == "" {
			streamName = "TELLER"
		}

		natsPubSub, err := wpubsub.NewNATSPubSub(natsURL, streamName, logger)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create NATS pubsub: %w", err)
		}
		ps = natsPubSub
		cleanup = func() {
			if err := natsPubSub.Close(); err != nil {
				log.Errorf("Failed to close NATS connection: %v", err)
			}
		}
		log.Infof("Using NATS pubsub with URL: %s", natsURL)

	case "memory":
		// Create in-memory pubsub for development
		memPubSub := wpubsub.NewMemoryPubSub(logger)
		ps = memPubSub
		cleanup = func() {
			if err := memPubSub.Close(); err != nil {
				log.Errorf("Failed to close memory pubsub: %v", err)
			}
		}
		log.Info("Using in-memory pubsub")

	case "kafka":
		// TODO: Implement Kafka pubsub
		return nil, nil, fmt.Errorf("kafka pubsub not yet implemented")

	case "redis":
		// TODO: Implement Redis pubsub
		return nil, nil, fmt.Errorf("redis pubsub not yet implemented")

	default:
		return nil, nil, fmt.Errorf("unknown pubsub type: %s", cfg.Type)
	}

	return ps, cleanup, nil
}