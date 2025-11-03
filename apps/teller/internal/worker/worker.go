package worker

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/biz"
)

// Worker represents the notification worker service
type Worker struct {
	notificationWorker *biz.NotificationWorker
	log                *log.Helper
}

// NewWorker creates a new worker instance
func NewWorker(
	notificationWorker *biz.NotificationWorker,
	logger log.Logger,
) *Worker {
	return &Worker{
		notificationWorker: notificationWorker,
		log:                log.NewHelper(log.With(logger, "module", "worker")),
	}
}

// Start starts the worker service
func (w *Worker) Start(ctx context.Context) error {
	w.log.Info("Starting worker service...")
	return w.notificationWorker.Start(ctx)
}

// Stop stops the worker service
func (w *Worker) Stop() error {
	w.log.Info("Stopping worker service...")
	w.notificationWorker.Stop()
	return nil
}