package data

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/biz"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/constant"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/data/entity"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/errors"
	"github.com/toeydevelopment/telltheworld/pkg/wpubsub"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var _ biz.NotificationRepo = (*notificationRepo)(nil)

type notificationRepo struct {
	data   *Data
	pubsub wpubsub.PubSub
	log    *log.Helper
}

// BulkSave implements biz.NotificationRepo.
func (r *notificationRepo) BulkSave(ctx context.Context, notifications []*entity.Notification) error {
	return r.data.db.WithContext(ctx).CreateInBatches(notifications, 1000).Error
}

func NewNotificationRepo(data *Data, pubsub wpubsub.PubSub, logger log.Logger) *notificationRepo {
	return &notificationRepo{
		data:   data,
		pubsub: pubsub,
		log:    log.NewHelper(log.With(logger, "module", "data/notification")),
	}
}

func (r *notificationRepo) Save(ctx context.Context, notification *entity.Notification) error {
	return r.data.db.WithContext(ctx).Create(notification).Error
}

func (r *notificationRepo) GetByNotificationID(ctx context.Context, notificationID string) (*entity.Notification, error) {
	var notification entity.Notification
	err := r.data.db.WithContext(ctx).Where("notification_id = ?", notificationID).First(&notification).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.ErrNotificationNotFound
		}
		return nil, err
	}
	return &notification, nil
}

func (r *notificationRepo) UpdateStatus(ctx context.Context, notificationID string, status constant.NotificationStatus) error {
	return r.data.db.WithContext(ctx).Model(&entity.Notification{}).
		Where("notification_id = ?", notificationID).
		Update("status", status).Error
}

func (r *notificationRepo) UpdateChannelResult(ctx context.Context, notificationID string, channelResult entity.NotificationStatusPerChannel) error {
	// First, get the notification
	var notification entity.Notification
	if err := r.data.db.WithContext(ctx).Where("notification_id = ?", notificationID).First(&notification).Error; err != nil {
		return err
	}

	// Update the results array
	results := notification.GetStatusPerChannel()

	// Check if this channel already has a result and update it
	found := false
	for i, result := range results {
		if result.Channel == channelResult.Channel {
			results[i] = channelResult
			found = true
			break
		}
	}

	// If not found, append the new result
	if !found {
		results = append(results, channelResult)
	}

	notification.Results = datatypes.NewJSONSlice(results)

	// Determine overall status based on all channel results
	allCompleted := true
	anyFailed := false
	anySent := false

	for _, result := range results {
		if result.Status != constant.Sent && result.Status != constant.Failed {
			allCompleted = false
		}
		if result.Status == constant.Failed {
			anyFailed = true
		}
		if result.Status == constant.Sent {
			anySent = true
		}
	}

	// Update notification status based on results
	if allCompleted {
		if anyFailed && !anySent {
			notification.Status = constant.Failed
		} else if anySent {
			notification.Status = constant.Sent
		}
	}

	// Update sent time if status is sent
	if notification.Status == constant.Sent && notification.SentAt == nil {
		now := time.Now()
		notification.SentAt = &now
	}

	// Save the updated notification
	return r.data.db.WithContext(ctx).Save(&notification).Error
}

func (r *notificationRepo) UpdateNotificationWithResults(ctx context.Context, notification *entity.Notification) error {
	return r.data.db.WithContext(ctx).Save(notification).Error
}

// ClaimNotificationForProcessing atomically claims a notification for processing by a worker
func (r *notificationRepo) ClaimNotificationForProcessing(ctx context.Context, notificationID string, workerID string) (*entity.Notification, error) {
	var notification entity.Notification

	// Use UPDATE ... RETURNING for atomic claim operation
	// This is simpler and more efficient than SELECT FOR UPDATE
	sql := `
		UPDATE notifications
		SET status = ?,
		    processing_worker_id = ?,
		    processing_started_at = ?,
		    processing_attempts = processing_attempts + 1,
		    updated_at = ?
		WHERE notification_id = ?
		  AND status = ?
		RETURNING *
	`

	now := time.Now()
	err := r.data.db.WithContext(ctx).Raw(sql,
		constant.Processing,   // new status
		workerID,              // processing_worker_id
		now,                   // processing_started_at
		now,                   // updated_at
		notificationID,        // notification_id
		constant.Pending,      // current status must be pending
	).Scan(&notification).Error

	if err != nil {
		return nil, err
	}

	// Check if any row was updated
	if notification.ID == 0 {
		return nil, errors.ErrNotificationNotPending
	}

	r.log.Infof("Worker %s claimed notification %s", workerID, notificationID)
	return &notification, nil
}

// ReleaseNotificationFromProcessing releases a notification from processing state
func (r *notificationRepo) ReleaseNotificationFromProcessing(ctx context.Context, notificationID string, workerID string, status constant.NotificationStatus) error {
	// Use raw SQL for clarity and atomic update
	sql := `
		UPDATE notifications
		SET status = ?,
		    processing_worker_id = NULL,
		    processing_started_at = NULL,
		    sent_at = CASE WHEN ? = ? THEN NOW() ELSE sent_at END,
		    updated_at = NOW()
		WHERE notification_id = ?
		  AND processing_worker_id = ?
	`

	result := r.data.db.WithContext(ctx).Exec(sql,
		status,               // new status
		status,               // for CASE condition
		constant.Sent,        // check if status is sent
		notificationID,       // notification_id
		workerID,            // must be owned by this worker
	)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.ErrNotificationNotOwned
	}

	r.log.Infof("Worker %s released notification %s with status %s", workerID, notificationID, status)
	return nil
}

// ResetStuckNotifications resets notifications that have been processing for too long
func (r *notificationRepo) ResetStuckNotifications(ctx context.Context, stuckAfter time.Duration) (int64, error) {
	// Use raw SQL for atomic batch update
	sql := `
		UPDATE notifications
		SET status = ?,
		    processing_worker_id = NULL,
		    processing_started_at = NULL,
		    updated_at = NOW()
		WHERE status = ?
		  AND processing_started_at < ?
		  AND processing_attempts < 5  -- Prevent infinite retries
	`

	cutoffTime := time.Now().Add(-stuckAfter)
	result := r.data.db.WithContext(ctx).Exec(sql,
		constant.Pending,     // reset to pending
		constant.Processing,  // current status
		cutoffTime,          // stuck threshold
	)

	if result.Error != nil {
		return 0, result.Error
	}

	if result.RowsAffected > 0 {
		r.log.Infof("Reset %d stuck notifications", result.RowsAffected)
	}

	return result.RowsAffected, nil
}

// PublishNotification publishes a notification to the event broker
func (r *notificationRepo) PublishNotification(ctx context.Context, notification *entity.Notification) error {
	// Convert entity channels to pubsub channels
	channels := notification.GetChannelDestinations()
	pubsubChannels := make([]wpubsub.ChannelDestination, 0, len(channels))

	for _, ch := range channels {
		pubsubChannels = append(pubsubChannels, wpubsub.ChannelDestination{
			Channel:     string(ch.Channel),
			Destination: ch.Destination,
		})
	}

	// Create pubsub message
	pubsubMsg := &wpubsub.NotificationMessage{
		NotificationID: notification.NotificationID,
		Title:          notification.Title,
		Body:           notification.Body,
		RefID:          notification.RefID,
		Channels:       pubsubChannels,
		Metadata: map[string]string{
			"notification_id": notification.NotificationID,
			"ref_id":          notification.RefID,
		},
	}

	// Add schedule time if present
	if notification.ScheduleAt != nil {
		scheduleUnix := notification.ScheduleAt.Unix()
		pubsubMsg.ScheduleAt = &scheduleUnix
	}

	// Serialize and publish
	data, err := json.Marshal(pubsubMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal notification message: %w", err)
	}

	message := &wpubsub.Message{
		Topic: "notifications.pending",
		Key:   notification.NotificationID,
		Value: data,
		Metadata: map[string]string{
			"notification_id": notification.NotificationID,
			"ref_id":          notification.RefID,
		},
	}

	if err := r.pubsub.Publish(ctx, message); err != nil {
		return fmt.Errorf("failed to publish notification: %w", err)
	}

	r.log.Debugf("Published notification %s to event broker", notification.NotificationID)
	return nil
}

// SubscribeToNotifications subscribes to notifications from the event broker
func (r *notificationRepo) SubscribeToNotifications(ctx context.Context, handler biz.NotificationHandler) error {
	// Create a wrapper handler that converts wpubsub.Message to entity.Notification
	wrapperHandler := func(ctx context.Context, msg *wpubsub.Message) error {
		// Parse the notification message
		var notificationMsg wpubsub.NotificationMessage
		if err := json.Unmarshal(msg.Value, &notificationMsg); err != nil {
			r.log.Errorf("Failed to unmarshal notification message: %v", err)
			return err
		}

		// Fetch the full notification from the database
		notification, err := r.GetByNotificationID(ctx, notificationMsg.NotificationID)
		if err != nil {
			r.log.Errorf("Failed to fetch notification %s: %v", notificationMsg.NotificationID, err)
			return err
		}

		// Call the business logic handler with the notification entity
		return handler(ctx, notification)
	}

	// Subscribe to the pending notifications topic
	return r.pubsub.Subscribe(ctx, "notifications.pending", wrapperHandler)
}
