package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	v1 "github.com/toeydevelopment/telltheworld/apis/teller/v1"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/constant"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/data/entity"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

// NotificationHandler is a callback for processing notifications from pubsub
type NotificationHandler func(ctx context.Context, notification *entity.Notification) error

type NotificationRepo interface {
	Save(ctx context.Context, notification *entity.Notification) error
	BulkSave(ctx context.Context, notifications []*entity.Notification) error
	GetByNotificationID(ctx context.Context, notificationID string) (*entity.Notification, error)
	UpdateStatus(ctx context.Context, notificationID string, status constant.NotificationStatus) error
	UpdateChannelResult(ctx context.Context, notificationID string, channelResult entity.NotificationStatusPerChannel) error
	UpdateNotificationWithResults(ctx context.Context, notification *entity.Notification) error

	// Atomic operations for worker processing
	ClaimNotificationForProcessing(ctx context.Context, notificationID string, workerID string) (*entity.Notification, error)
	ReleaseNotificationFromProcessing(ctx context.Context, notificationID string, workerID string, status constant.NotificationStatus) error
	ResetStuckNotifications(ctx context.Context, stuckAfter time.Duration) (int64, error)

	// PubSub operations
	PublishNotification(ctx context.Context, notification *entity.Notification) error
	SubscribeToNotifications(ctx context.Context, handler NotificationHandler) error
}

type Notification struct {
	repo NotificationRepo
	log  *log.Helper
}

func NewNotification(repo NotificationRepo, logger log.Logger) *Notification {
	return &Notification{
		repo: repo,
		log:  log.NewHelper(log.With(logger, "module", "biz/notification")),
	}
}

// prepareNotification creates a notification entity from a request
func (n *Notification) prepareNotification(req *v1.SendToUserRequest) *entity.Notification {
	notificationID := uuid.New().String()

	channelDestinations := make([]entity.NotificationChannelDestination, 0, len(req.Destinations))

	for _, dest := range req.Destinations {
		channel := convertProtoChannelToConstant(dest.Channel)
		channelDestinations = append(channelDestinations, entity.NotificationChannelDestination{
			Channel:     channel,
			Destination: dest.Destination,
		})
	}

	notification := &entity.Notification{
		NotificationID: notificationID,
		Title:          req.Notification.Title,
		Body:           req.Notification.Body,
		RefID:          req.Notification.RefId,
		Status:         constant.Pending,
		Channels:       datatypes.NewJSONSlice(channelDestinations),
		Results:        datatypes.NewJSONSlice([]entity.NotificationStatusPerChannel{}),
	}

	if req.Notification.ScheduleAt != nil {
		scheduleTime := req.Notification.ScheduleAt.AsTime()
		notification.ScheduleAt = &scheduleTime
	}

	return notification
}

// SendToUser sends a notification to specific user channels
func (n *Notification) SendToUser(ctx context.Context, req *v1.SendToUserRequest) (string, error) {
	notification := n.prepareNotification(req)

	if err := n.repo.Save(ctx, notification); err != nil {
		n.log.Errorf("Failed to save notification: %v", err)

		return "", errors.InternalServer("INTERNAL", fmt.Sprintf("failed to save notification: %v", err))
	}

	if err := n.repo.PublishNotification(ctx, notification); err != nil {
		n.log.Errorf("Failed to publish notification to event broker: %v", err)
		_ = n.repo.UpdateStatus(ctx, notification.NotificationID, constant.Failed)
		return "", errors.InternalServer("INTERNAL", fmt.Sprintf("failed to publish notification: %v", err))
	}

	n.log.Infof("Notification %s created and published successfully", notification.NotificationID)
	return notification.NotificationID, nil
}

// BulkSendToUser sends multiple notifications
func (n *Notification) BulkSendToUser(ctx context.Context, req *v1.BulkSendToUserRequest) ([]string, error) {
	if len(req.Requests) == 0 {
		return []string{}, nil
	}

	notifications := make([]*entity.Notification, 0, len(req.Requests))
	notificationIDs := make([]string, 0, len(req.Requests))

	for _, sendReq := range req.Requests {
		notification := n.prepareNotification(sendReq)
		notifications = append(notifications, notification)
		notificationIDs = append(notificationIDs, notification.NotificationID)
	}

	if err := n.repo.BulkSave(ctx, notifications); err != nil {
		n.log.Errorf("Failed to bulk save notifications: %v", err)
		return nil, errors.InternalServer("INTERNAL", fmt.Sprintf("failed to bulk save notifications: %v", err))
	}

	failedPublishes := []int{}
	for i, notification := range notifications {
		// We can enhance this to use a batch publish later
		if err := n.repo.PublishNotification(ctx, notification); err != nil {
			n.log.Errorf("Failed to publish notification %s to event broker: %v", notification.NotificationID, err)
			failedPublishes = append(failedPublishes, i)
			// Update status to failed if publish fails
			_ = n.repo.UpdateStatus(ctx, notification.NotificationID, constant.Failed)
		}
	}

	// If some publishes failed, set their IDs to empty string
	for _, idx := range failedPublishes {
		notificationIDs[idx] = ""
	}

	n.log.Infof("Bulk created %d notifications, %d publish failures", len(notifications), len(failedPublishes))
	return notificationIDs, nil
}

// GetNotificationStatus retrieves the status of a notification
func (n *Notification) GetNotificationStatus(ctx context.Context, notificationID string) ([]*v1.StatusPerChannel, error) {
	notification, err := n.repo.GetByNotificationID(ctx, notificationID)
	if err != nil {
		return nil, errors.BadRequest("BAD", err.Error())
	}

	statuses := make([]*v1.StatusPerChannel, 0)
	for _, result := range notification.GetStatusPerChannel() {
		status := &v1.StatusPerChannel{
			Channel: convertConstantChannelToProto(result.Channel),
			Status:  convertConstantStatusToProto(result.Status),
		}

		if result.FailureReason != "" {
			status.FailReason = &result.FailureReason
		}

		if result.CompletedAt != nil {
			status.CompletedAt = timestamppb.New(*result.CompletedAt)
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// Helper functions to convert between proto and constant types
func convertProtoChannelToConstant(channel v1.NotificationChannel) constant.NotificationChannel {
	switch channel {
	case v1.NotificationChannel_NotificationChannel_Push:
		return constant.Push
	case v1.NotificationChannel_NotificationChannel_Email:
		return constant.Email
	default:
		return constant.Email // Default fallback
	}
}

func convertConstantChannelToProto(channel constant.NotificationChannel) v1.NotificationChannel {
	switch channel {
	case constant.Push:
		return v1.NotificationChannel_NotificationChannel_Push
	case constant.Email:
		return v1.NotificationChannel_NotificationChannel_Email
	default:
		return v1.NotificationChannel_NotificationChannel_Unknown
	}
}

func convertConstantStatusToProto(status constant.NotificationStatus) v1.NotificationStatus {
	switch status {
	case constant.Pending:
		return v1.NotificationStatus_NotificationStatus_Pending
	case constant.Processing:
		return v1.NotificationStatus_NotificationStatus_Pending // Map processing to pending for API
	case constant.Sent:
		return v1.NotificationStatus_NotificationStatus_Sent
	case constant.Failed:
		return v1.NotificationStatus_NotificationStatus_Failed
	default:
		return v1.NotificationStatus_NotificationStatus_Unknown
	}
}
