package biz

import (
	"context"
	stderrors "errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/constant"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/data/entity"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/errors"
)

// NotificationWorker processes notifications from the event broker
type NotificationWorker struct {
	repo     NotificationRepo
	registry *ProviderRegistry
	log      *log.Helper
	wg       sync.WaitGroup
	workerID string
	stopCh   chan struct{}
}

// NewNotificationWorker creates a new notification worker
func NewNotificationWorker(
	repo NotificationRepo,
	registry *ProviderRegistry,
	logger log.Logger,
) *NotificationWorker {
	hostname, _ := os.Hostname()
	workerID := fmt.Sprintf("%s-%s", hostname, uuid.New().String()[:8])

	return &NotificationWorker{
		repo:     repo,
		registry: registry,
		log:      log.NewHelper(log.With(logger, "module", "biz/notification_worker", "worker_id", workerID)),
		workerID: workerID,
		stopCh:   make(chan struct{}),
	}
}

// Start starts the notification worker
func (w *NotificationWorker) Start(ctx context.Context) error {
	w.log.Info("Starting notification worker...")

	// Start cleanup goroutine for stuck notifications
	go w.cleanupStuckNotifications(ctx)

	err := w.repo.SubscribeToNotifications(ctx, w.processNotification)
	if err != nil {
		w.log.Errorf("Failed to subscribe to notifications: %v", err)
		return err
	}

	w.log.Info("Notification worker started successfully")

	<-ctx.Done()
	w.log.Info("Shutting down notification worker...")

	close(w.stopCh)

	w.wg.Wait()
	w.log.Info("Notification worker shut down successfully")

	return nil
}

// cleanupStuckNotifications periodically resets stuck notifications
func (w *NotificationWorker) cleanupStuckNotifications(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			// Reset notifications that have been processing for more than 5 minutes
			count, err := w.repo.ResetStuckNotifications(ctx, 5*time.Minute)
			if err != nil {
				w.log.Errorf("Failed to reset stuck notifications: %v", err)
			} else if count > 0 {
				w.log.Infof("Reset %d stuck notifications", count)
			}
		}
	}
}

// processNotification processes a single notification
func (w *NotificationWorker) processNotification(ctx context.Context, notification *entity.Notification) error {
	w.wg.Add(1)
	defer w.wg.Done()

	w.log.Infof("Processing notification: %s", notification.NotificationID)

	// Check if notification should be scheduled for later
	// We can to have another process to seek a database to find the notifications that are scheduled for later and process them.
	if notification.ScheduleAt != nil && notification.ScheduleAt.After(time.Now()) {
		w.log.Infof("Notification %s is scheduled for %v, skipping for now",
			notification.NotificationID, notification.ScheduleAt)
		return nil
	}

	// Atomically claim the notification for processing
	claimedNotification, err := w.repo.ClaimNotificationForProcessing(ctx, notification.NotificationID, w.workerID)
	if err != nil {
		if stderrors.Is(err, errors.ErrNotificationNotPending) {
			w.log.Infof("Notification %s already being processed by another worker, skipping",
				notification.NotificationID)
			return nil
		}
		w.log.Errorf("Failed to claim notification %s: %v", notification.NotificationID, err)
		return err
	}

	w.log.Infof("Successfully claimed notification %s for processing", claimedNotification.NotificationID)

	// Ensure we release the notification after processing
	defer func() {
		// Determine final status based on results
		finalStatus := w.determineFinalStatus(claimedNotification)
		if err := w.repo.ReleaseNotificationFromProcessing(ctx, claimedNotification.NotificationID, w.workerID, finalStatus); err != nil {
			w.log.Errorf("Failed to release notification %s: %v", claimedNotification.NotificationID, err)
		} else {
			w.log.Infof("Released notification %s with status: %s", claimedNotification.NotificationID, finalStatus)
		}
	}()

	// Process each channel concurrently
	var wg sync.WaitGroup
	for _, channelDest := range claimedNotification.GetChannelDestinations() {
		wg.Add(1)
		go func(channel constant.NotificationChannel, destination string) {
			defer wg.Done()
			w.sendToChannel(ctx, claimedNotification, channel, destination)
		}(channelDest.Channel, channelDest.Destination)
	}

	wg.Wait()

	w.log.Infof("Completed processing notification: %s", claimedNotification.NotificationID)
	return nil
}

// determineFinalStatus determines the final status based on channel results
func (w *NotificationWorker) determineFinalStatus(notification *entity.Notification) constant.NotificationStatus {
	// Get the latest notification state
	latestNotification, err := w.repo.GetByNotificationID(context.Background(), notification.NotificationID)
	if err != nil {
		return constant.Failed
	}

	results := latestNotification.GetStatusPerChannel()
	if len(results) == 0 {
		return constant.Failed
	}

	anySent := false
	allFailed := true

	for _, result := range results {
		if result.Status == constant.Sent {
			anySent = true
			allFailed = false
			break
		}
		if result.Status != constant.Failed {
			allFailed = false
		}
	}

	if anySent {
		return constant.Sent
	}
	if allFailed {
		return constant.Failed
	}
	return constant.Pending
}

// sendToChannel sends a notification to a specific channel
func (w *NotificationWorker) sendToChannel(
	ctx context.Context,
	notification *entity.Notification,
	channel constant.NotificationChannel,
	destination string,
) {
	w.log.Infof("Sending notification %s to %s channel: %s",
		notification.NotificationID, channel, destination)

	provider, err := w.registry.GetProvider(channel)
	if err != nil {
		w.log.Errorf("Failed to get provider for channel %s: %v", channel, err)
		w.updateChannelStatus(ctx, notification.NotificationID, channel, constant.Failed, err.Error())
		return
	}

	// Send the notification
	startTime := time.Now()
	err = provider.Send(ctx, destination, notification.Title, notification.Body, nil)
	completedAt := time.Now()

	if err != nil {
		w.log.Errorf("Failed to send notification %s to %s: %v",
			notification.NotificationID, destination, err)
		w.updateChannelStatus(ctx, notification.NotificationID, channel, constant.Failed, err.Error())
	} else {
		w.log.Infof("Successfully sent notification %s to %s via %s (took %v)",
			notification.NotificationID, destination, channel, completedAt.Sub(startTime))
		w.updateChannelStatus(ctx, notification.NotificationID, channel, constant.Sent, "")
	}
}

// updateChannelStatus updates the status of a notification channel
func (w *NotificationWorker) updateChannelStatus(
	ctx context.Context,
	notificationID string,
	channel constant.NotificationChannel,
	status constant.NotificationStatus,
	failureReason string,
) {
	completedAt := time.Now()
	channelResult := entity.NotificationStatusPerChannel{
		Channel:       channel,
		Status:        status,
		FailureReason: failureReason,
		CompletedAt:   &completedAt,
	}

	if err := w.repo.UpdateChannelResult(ctx, notificationID, channelResult); err != nil {
		w.log.Errorf("Failed to update channel result for notification %s: %v",
			notificationID, err)
	}
}

// Stop gracefully stops the worker
func (w *NotificationWorker) Stop() {
	w.log.Info("Stopping notification worker...")
}
