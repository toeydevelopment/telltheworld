package data

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	tellerv1 "github.com/toeydevelopment/telltheworld/apis/teller/v1"
	"github.com/toeydevelopment/telltheworld/internal/wkratos"

	"github.com/toeydevelopment/telltheworld/apps/ecommerce/internal/biz"
	"github.com/toeydevelopment/telltheworld/apps/ecommerce/internal/conf"
)

var _ biz.EcommerceRepo = (*commerceRepo)(nil)

type commerceRepo struct {
	client tellerv1.NotificationServiceClient
	log    *log.Helper
}

// NewCommerceRepo creates a new teller client
func NewCommerceRepo(c *conf.TellerClient, logger log.Logger) (*commerceRepo, func(), error) {
	log := log.NewHelper(log.With(logger, "module", "data/teller_client"))

	// Default endpoint if not configured
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = "localhost:9000"
	}

	// Create gRPC connection

	log.Infof("Attempting to connect to teller service at %s", endpoint)
	conn, err := wkratos.NewClient(wkratos.ClientOption{
		Addr:    endpoint,
		Timeout: c.GetTimeout().AsDuration(),
	}, nil)
	if err != nil {
		return nil, nil, err
	}
	client := tellerv1.NewNotificationServiceClient(conn)

	log.Infof("Connected to teller service at %s", endpoint)

	return &commerceRepo{
		client: client,
		log:    log,
	}, func() {}, nil
}

// SendNotification sends a notification via the teller service
func (tc *commerceRepo) SendNotification(ctx context.Context, event *biz.Event) (string, error) {
	// Map event to notification request
	req := &tellerv1.SendToUserRequest{
		Notification: &tellerv1.Notification{
			Title: tc.getNotificationTitle(event),
			Body:  event.Message,
			RefId: tc.generateRefID(event),
		},
		Destinations: tc.getDestinations(event),
	}

	tc.log.Debugf("Sending notification via teller service: %+v", req)

	resp, err := tc.client.SendToUser(ctx, req)
	if err != nil {
		return "", fmt.Errorf("teller service error: %w", err)
	}

	tc.log.Infof("Notification sent successfully: %s", resp.NotificationId)
	return resp.NotificationId, nil
}

func (tc *commerceRepo) getNotificationTitle(event *biz.Event) string {
	switch event.Type {
	case biz.EventTypeChatMessage:
		return "New Message"
	case biz.EventTypePurchase:
		return "New Purchase!"
	case biz.EventTypePaymentReminder:
		return "Payment Reminder"
	case biz.EventTypeShippingUpdate:
		return "Shipping Update"
	default:
		return "Notification"
	}
}

func (tc *commerceRepo) generateRefID(event *biz.Event) string {
	// Generate unique reference ID based on event type
	prefix := ""
	switch event.Type {
	case biz.EventTypeChatMessage:
		prefix = "CHAT"
	case biz.EventTypePurchase:
		prefix = "PURCHASE"
	case biz.EventTypePaymentReminder:
		prefix = "PAYMENT"
	case biz.EventTypeShippingUpdate:
		prefix = "SHIPPING"
	}

	if event.OrderID != "" {
		return fmt.Sprintf("%s-%s-%d", prefix, event.OrderID, time.Now().Unix())
	}
	return fmt.Sprintf("%s-%s", prefix, uuid.New().String()[:8])
}

func (tc *commerceRepo) getDestinations(event *biz.Event) []*tellerv1.ChannelDestination {
	destinations := []*tellerv1.ChannelDestination{}

	switch event.Type {
	case biz.EventTypeChatMessage, biz.EventTypePurchase:
		// Send to seller via email and push
		if event.SellerID != "" {
			destinations = append(destinations,
				&tellerv1.ChannelDestination{
					Channel:     tellerv1.NotificationChannel_NotificationChannel_Email,
					Destination: fmt.Sprintf("seller_%s@example.com", event.SellerID),
				},
				&tellerv1.ChannelDestination{
					Channel:     tellerv1.NotificationChannel_NotificationChannel_Push,
					Destination: fmt.Sprintf("device_seller_%s", event.SellerID),
				},
			)
		}

	case biz.EventTypePaymentReminder, biz.EventTypeShippingUpdate:
		// Send to buyer via push only
		if event.BuyerID != "" {
			destinations = append(destinations,
				&tellerv1.ChannelDestination{
					Channel:     tellerv1.NotificationChannel_NotificationChannel_Push,
					Destination: fmt.Sprintf("device_buyer_%s", event.BuyerID),
				},
			)
		}
	}

	// Fallback if no destinations
	if len(destinations) == 0 {
		destinations = append(destinations, &tellerv1.ChannelDestination{
			Channel:     tellerv1.NotificationChannel_NotificationChannel_Email,
			Destination: "admin@example.com",
		})
	}

	return destinations
}
