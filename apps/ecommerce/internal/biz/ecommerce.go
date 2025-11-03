package biz

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

type EcommerceRepo interface {
	SendNotification(ctx context.Context, event *Event) (string, error)
}

// Event represents an ecommerce event
type Event struct {
	Type     EventType
	BuyerID  string
	SellerID string
	OrderID  string
	Message  string
	Metadata map[string]string
}

// EventType represents the type of ecommerce event
type EventType int

const (
	EventTypeChatMessage EventType = iota
	EventTypePurchase
	EventTypePaymentReminder
	EventTypeShippingUpdate
)

// EcommerceUsecase is an Ecommerce usecase.
type EcommerceUsecase struct {
	repo EcommerceRepo
	log  *log.Helper
}

// NewEcommerceUsecase new an Ecommerce usecase.
func NewEcommerceUsecase(repo EcommerceRepo, logger log.Logger) *EcommerceUsecase {
	return &EcommerceUsecase{
		repo: repo,
		log:  log.NewHelper(log.With(logger, "module", "biz/ecommerce")),
	}
}

// SendChatMessage handles chat message event
func (uc *EcommerceUsecase) SendChatMessage(ctx context.Context, buyerID, sellerID, message, conversationID string) (string, error) {
	uc.log.WithContext(ctx).Infof("Processing chat message from buyer %s to seller %s", buyerID, sellerID)

	event := &Event{
		Type:     EventTypeChatMessage,
		BuyerID:  buyerID,
		SellerID: sellerID,
		Message:  fmt.Sprintf("New message from buyer: %s", message),
		Metadata: map[string]string{
			"conversation_id": conversationID,
			"message":         message,
			"event_type":      "chat_message",
		},
	}

	notificationID, err := uc.repo.SendNotification(ctx, event)
	if err != nil {
		return "", fmt.Errorf("failed to send chat notification: %w", err)
	}

	return notificationID, nil
}

// TriggerPurchase handles purchase event
func (uc *EcommerceUsecase) TriggerPurchase(ctx context.Context, buyerID, sellerID, orderID string, amount float64, itemCount int) (string, error) {
	uc.log.WithContext(ctx).Infof("Processing purchase order %s from buyer %s to seller %s", orderID, buyerID, sellerID)

	event := &Event{
		Type:     EventTypePurchase,
		BuyerID:  buyerID,
		SellerID: sellerID,
		OrderID:  orderID,
		Message:  fmt.Sprintf("New purchase! Order #%s - Amount: $%.2f - Items: %d", orderID, amount, itemCount),
		Metadata: map[string]string{
			"order_id":   orderID,
			"amount":     fmt.Sprintf("%.2f", amount),
			"item_count": fmt.Sprintf("%d", itemCount),
			"event_type": "purchase",
		},
	}

	notificationID, err := uc.repo.SendNotification(ctx, event)
	if err != nil {
		return "", fmt.Errorf("failed to send purchase notification: %w", err)
	}

	return notificationID, nil
}

// SendPaymentReminder handles payment reminder event
func (uc *EcommerceUsecase) SendPaymentReminder(ctx context.Context, buyerID, orderID string, amountDue float64, dueDate string) (string, error) {
	uc.log.WithContext(ctx).Infof("Processing payment reminder for order %s to buyer %s", orderID, buyerID)

	event := &Event{
		Type:    EventTypePaymentReminder,
		BuyerID: buyerID,
		OrderID: orderID,
		Message: fmt.Sprintf("Payment reminder: $%.2f due by %s for Order #%s", amountDue, dueDate, orderID),
		Metadata: map[string]string{
			"order_id":   orderID,
			"amount_due": fmt.Sprintf("%.2f", amountDue),
			"due_date":   dueDate,
			"event_type": "payment_reminder",
		},
	}

	notificationID, err := uc.repo.SendNotification(ctx, event)
	if err != nil {
		return "", fmt.Errorf("failed to send payment reminder: %w", err)
	}

	return notificationID, nil
}

// SendShippingUpdate handles shipping update event
func (uc *EcommerceUsecase) SendShippingUpdate(ctx context.Context, buyerID, orderID, trackingNumber, carrier, status string) (string, error) {
	uc.log.WithContext(ctx).Infof("Processing shipping update for order %s to buyer %s", orderID, buyerID)

	statusMessage := "Your order has been "
	switch status {
	case "shipped":
		statusMessage += "shipped"
	case "in_transit":
		statusMessage = "Your order is in transit"
	case "delivered":
		statusMessage = "Your order has been delivered"
	default:
		statusMessage += status
	}

	event := &Event{
		Type:    EventTypeShippingUpdate,
		BuyerID: buyerID,
		OrderID: orderID,
		Message: fmt.Sprintf("%s! Order #%s - Tracking: %s (%s)", statusMessage, orderID, trackingNumber, carrier),
		Metadata: map[string]string{
			"order_id":        orderID,
			"tracking_number": trackingNumber,
			"carrier":         carrier,
			"status":          status,
			"event_type":      "shipping_update",
		},
	}

	notificationID, err := uc.repo.SendNotification(ctx, event)
	if err != nil {
		return "", fmt.Errorf("failed to send shipping update: %w", err)
	}

	return notificationID, nil
}
