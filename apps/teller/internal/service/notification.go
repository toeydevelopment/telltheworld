package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	v1 "github.com/toeydevelopment/telltheworld/apis/teller/v1"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/biz"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ v1.NotificationServiceServer = (*Notification)(nil)

type Notification struct {
	v1.UnimplementedNotificationServiceServer
	biz *biz.Notification
	log *log.Helper
}

func NewNotification(biz *biz.Notification, logger log.Logger) *Notification {
	return &Notification{
		biz: biz,
		log: log.NewHelper(log.With(logger, "module", "service/notification")),
	}
}

// Boardcast implements v1.NotificationServiceServer.
func (n *Notification) Boardcast(ctx context.Context, req *v1.BoardcastRequest) (*emptypb.Empty, error) {
	// To be implemented later
	n.log.Info("Boardcast called but not yet implemented")
	return &emptypb.Empty{}, nil
}

// BulkSendToUser implements v1.NotificationServiceServer.
func (n *Notification) BulkSendToUser(ctx context.Context, req *v1.BulkSendToUserRequest) (*v1.BulkSendToUserReply, error) {
	n.log.Infof("BulkSendToUser called with %d requests", len(req.Requests))

	notificationIDs, err := n.biz.BulkSendToUser(ctx, req)
	if err != nil {
		n.log.Errorf("Failed to bulk send notifications: %v", err)
		return nil, err
	}

	replies := make([]*v1.WillPushReply, 0, len(notificationIDs))
	for _, id := range notificationIDs {
		if id != "" {
			replies = append(replies, &v1.WillPushReply{
				NotificationId: id,
			})
		}
	}

	return &v1.BulkSendToUserReply{
		Replies: replies,
	}, nil
}

// GetNotificationStatus implements v1.NotificationServiceServer.
func (n *Notification) GetNotificationStatus(ctx context.Context, req *v1.GetNotificationStatusRequest) (*v1.GetNotificationStatusReply, error) {
	n.log.Infof("GetNotificationStatus called for notification ID: %s", req.NotificationId)

	statuses, err := n.biz.GetNotificationStatus(ctx, req.NotificationId)
	if err != nil {
		n.log.Errorf("Failed to get notification status: %v", err)
		return nil, err
	}

	// Convert biz layer response to proto response
	replyStatuses := make([]*v1.GetNotificationStatusReply_StatusPerChannel, 0, len(statuses))
	for _, status := range statuses {
		replyStatus := &v1.GetNotificationStatusReply_StatusPerChannel{
			Channel:     status.Channel,
			Status:      status.Status,
			FailReason:  status.FailReason,
			CompletedAt: status.CompletedAt,
		}
		replyStatuses = append(replyStatuses, replyStatus)
	}

	return &v1.GetNotificationStatusReply{
		Statuses: replyStatuses,
	}, nil
}

// SendToUser implements v1.NotificationServiceServer.
func (n *Notification) SendToUser(ctx context.Context, req *v1.SendToUserRequest) (*v1.WillPushReply, error) {
	n.log.Infof("SendToUser called with title: %s, refID: %s", req.Notification.Title, req.Notification.RefId)

	notificationID, err := n.biz.SendToUser(ctx, req)
	if err != nil {
		n.log.Errorf("Failed to send notification: %v", err)
		return nil, err
	}

	return &v1.WillPushReply{
		NotificationId: notificationID,
	}, nil
}
