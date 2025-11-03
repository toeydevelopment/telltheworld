package entity

import (
	"time"

	"github.com/toeydevelopment/telltheworld/apps/teller/internal/constant"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type NotificationChannelDestination struct {
	Channel     constant.NotificationChannel
	Destination string
}

type NotificationStatusPerChannel struct {
	Channel       constant.NotificationChannel
	Status        constant.NotificationStatus
	FailureReason string
	CompletedAt   *time.Time
}

type Notification struct {
	gorm.Model
	NotificationID        string                                                `gorm:"uniqueIndex;not null"` // UUID for the notification
	Title                 string                                                `gorm:"not null"`
	Body                  string                                                `gorm:"type:text"`
	RefID                 string                                                `gorm:"index"` // Reference ID from the request
	ScheduleAt            *time.Time                                            // When to send the notification
	SentAt                *time.Time                                            // When the notification was actually sent
	Channels              datatypes.JSONSlice[NotificationChannelDestination]   // Target channels and destinations
	Results               datatypes.JSONSlice[NotificationStatusPerChannel]     // Status per channel
	Status                constant.NotificationStatus                          `gorm:"default:'pending';index"`
	ProcessingStartedAt   *time.Time                                            `gorm:"index"` // When a worker started processing
	ProcessingWorkerID    string                                                // ID of the worker processing this notification
	ProcessingAttempts    int                                                   `gorm:"default:0"` // Number of processing attempts
}

func (Notification) TableName() string {
	return "notifications"
}

func (n Notification) GetChannelDestinations() []NotificationChannelDestination {
	return n.Channels
}

func (n Notification) GetStatusPerChannel() []NotificationStatusPerChannel {
	return n.Results
}
