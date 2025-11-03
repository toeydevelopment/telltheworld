package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/samber/lo"

	"github.com/toeydevelopment/telltheworld/apps/ecommerce/internal/biz"

	pb "github.com/toeydevelopment/telltheworld/apis/ecommerce/v1"
)

type EcommerceService struct {
	pb.UnimplementedEcommerceServiceServer
	uc  *biz.EcommerceUsecase
	log *log.Helper
}

func NewEcommerceService(uc *biz.EcommerceUsecase, logger log.Logger) *EcommerceService {
	return &EcommerceService{
		uc:  uc,
		log: log.NewHelper(log.With(logger, "module", "service/ecommerce")),
	}
}

func (s *EcommerceService) SendChatMessage(ctx context.Context, req *pb.SendChatMessageRequest) (*pb.SendChatMessageResponse, error) {
	s.log.WithContext(ctx).Infof("Received chat message request from buyer %s to seller %s", req.BuyerId, req.SellerId)

	notificationID, err := s.uc.SendChatMessage(ctx, req.BuyerId, req.SellerId, req.Message, req.ConversationId)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Failed to send chat message: %v", err)
		return &pb.SendChatMessageResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.SendChatMessageResponse{
		Success:        true,
		NotificationId: notificationID,
		Message:        "Chat notification sent successfully",
	}, nil
}

// TriggerPurchase handles purchase event
func (s *EcommerceService) TriggerPurchase(ctx context.Context, req *pb.TriggerPurchaseRequest) (*pb.TriggerPurchaseResponse, error) {
	s.log.WithContext(ctx).Infof("Received purchase request for order %s from buyer %s to seller %s",
		req.OrderId, req.BuyerId, req.SellerId)

	itemCount := lo.Reduce(req.GetItems(), func(acc int, item *pb.PurchaseItem, _ int) int {
		return acc + int(item.Quantity)
	}, 0)

	notificationID, err := s.uc.TriggerPurchase(ctx, req.BuyerId, req.SellerId, req.OrderId, req.Amount, itemCount)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Failed to send purchase notification: %v", err)
		return &pb.TriggerPurchaseResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.TriggerPurchaseResponse{
		Success:        true,
		NotificationId: notificationID,
		Message:        "Purchase notification sent successfully",
	}, nil
}

// SendPaymentReminder handles payment reminder event
func (s *EcommerceService) SendPaymentReminder(ctx context.Context, req *pb.SendPaymentReminderRequest) (*pb.SendPaymentReminderResponse, error) {
	s.log.WithContext(ctx).Infof("Received payment reminder request for order %s to buyer %s",
		req.OrderId, req.BuyerId)

	notificationID, err := s.uc.SendPaymentReminder(ctx, req.BuyerId, req.OrderId, req.AmountDue, req.DueDate)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Failed to send payment reminder: %v", err)
		return &pb.SendPaymentReminderResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.SendPaymentReminderResponse{
		Success:        true,
		NotificationId: notificationID,
		Message:        "Payment reminder sent successfully",
	}, nil
}

// SendShippingUpdate handles shipping update event
func (s *EcommerceService) SendShippingUpdate(ctx context.Context, req *pb.SendShippingUpdateRequest) (*pb.SendShippingUpdateResponse, error) {
	s.log.WithContext(ctx).Infof("Received shipping update for order %s to buyer %s with status %s",
		req.OrderId, req.BuyerId, req.Status)

	notificationID, err := s.uc.SendShippingUpdate(ctx, req.BuyerId, req.OrderId, req.TrackingNumber, req.Carrier, req.Status)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Failed to send shipping update: %v", err)
		return &pb.SendShippingUpdateResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.SendShippingUpdateResponse{
		Success:        true,
		NotificationId: notificationID,
		Message:        "Shipping update sent successfully",
	}, nil
}
