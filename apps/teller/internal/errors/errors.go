package errors

import "errors"

// Data layer errors
var (
	// ErrNotificationNotFound is returned when a notification is not found in the database
	ErrNotificationNotFound = errors.New("notification not found")

	// ErrNotificationNotPending is returned when trying to claim a notification that is not in pending status
	ErrNotificationNotPending = errors.New("notification not found or not in pending status")

	// ErrNotificationNotOwned is returned when trying to release a notification that is not owned by the worker
	ErrNotificationNotOwned = errors.New("notification not found or not owned by this worker")
)

// Business logic errors
var (
	// ErrProviderNotFound is returned when no provider is registered for a channel
	ErrProviderNotFound = errors.New("no provider registered for channel")
)
