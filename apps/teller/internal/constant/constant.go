package constant

type NotificationChannel string

const (
	Email NotificationChannel = "email"
	Push  NotificationChannel = "push"
)

type NotificationStatus string

const (
	Pending    NotificationStatus = "pending"
	Processing NotificationStatus = "processing"
	Sent       NotificationStatus = "sent"
	Failed     NotificationStatus = "failed"
)
