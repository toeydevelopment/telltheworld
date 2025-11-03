package biz

import (
	"context"
	stderrors "errors"
	"io"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/constant"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/data/entity"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/errors"
	"go.uber.org/mock/gomock"
	"gorm.io/datatypes"
)

func TestNewNotificationWorker(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockNotificationRepo(ctrl)
	logger := log.NewStdLogger(io.Discard)
	registry := NewProviderRegistry()

	worker := NewNotificationWorker(mockRepo, registry, logger)

	// Verify worker was created successfully
	assert.NotNil(t, worker)
}

func TestNotificationWorker_processNotification(t *testing.T) {
	tests := []struct {
		name      string
		notif     *entity.Notification
		setupMock func(*MockNotificationRepo, *MockNotificationProvider)
		wantErr   bool
	}{
		{
			name: "success - process single channel notification",
			notif: &entity.Notification{
				NotificationID: "notif-001",
				Title:          "Test Title",
				Body:           "Test Body",
				Status:         constant.Pending,
				Channels: datatypes.NewJSONSlice([]entity.NotificationChannelDestination{
					{Channel: constant.Email, Destination: "test@example.com"},
				}),
				Results: datatypes.NewJSONSlice([]entity.NotificationStatusPerChannel{}),
			},
			setupMock: func(repo *MockNotificationRepo, provider *MockNotificationProvider) {
				// Claim notification
				repo.EXPECT().
					ClaimNotificationForProcessing(gomock.Any(), "notif-001", gomock.Any()).
					Return(&entity.Notification{
						NotificationID: "notif-001",
						Title:          "Test Title",
						Body:           "Test Body",
						Status:         constant.Processing,
						Channels: datatypes.NewJSONSlice([]entity.NotificationChannelDestination{
							{Channel: constant.Email, Destination: "test@example.com"},
						}),
						Results: datatypes.NewJSONSlice([]entity.NotificationStatusPerChannel{}),
					}, nil)

				// Provider send
				provider.EXPECT().
					Send(gomock.Any(), "test@example.com", "Test Title", "Test Body", gomock.Any()).
					Return(nil)

				// Update channel result
				repo.EXPECT().
					UpdateChannelResult(gomock.Any(), "notif-001", gomock.Any()).
					Return(nil)

				// Get notification for final status determination
				repo.EXPECT().
					GetByNotificationID(gomock.Any(), "notif-001").
					Return(&entity.Notification{
						NotificationID: "notif-001",
						Status:         constant.Processing,
						Results: datatypes.NewJSONSlice([]entity.NotificationStatusPerChannel{
							{Channel: constant.Email, Status: constant.Sent},
						}),
					}, nil)

				// Release notification
				repo.EXPECT().
					ReleaseNotificationFromProcessing(gomock.Any(), "notif-001", gomock.Any(), constant.Sent).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - skip scheduled notification",
			notif: &entity.Notification{
				NotificationID: "notif-002",
				Title:          "Scheduled Title",
				Body:           "Scheduled Body",
				Status:         constant.Pending,
				ScheduleAt:     func() *time.Time { t := time.Now().Add(1 * time.Hour); return &t }(),
				Channels: datatypes.NewJSONSlice([]entity.NotificationChannelDestination{
					{Channel: constant.Email, Destination: "test@example.com"},
				}),
				Results: datatypes.NewJSONSlice([]entity.NotificationStatusPerChannel{}),
			},
			setupMock: func(repo *MockNotificationRepo, provider *MockNotificationProvider) {
				// No expectations since it should be skipped
			},
			wantErr: false,
		},
		{
			name: "success - already claimed by another worker",
			notif: &entity.Notification{
				NotificationID: "notif-003",
				Title:          "Test Title",
				Body:           "Test Body",
				Status:         constant.Pending,
				Channels: datatypes.NewJSONSlice([]entity.NotificationChannelDestination{
					{Channel: constant.Email, Destination: "test@example.com"},
				}),
			},
			setupMock: func(repo *MockNotificationRepo, provider *MockNotificationProvider) {
				// Claim fails - already claimed
				repo.EXPECT().
					ClaimNotificationForProcessing(gomock.Any(), "notif-003", gomock.Any()).
					Return(nil, errors.ErrNotificationNotPending)
			},
			wantErr: false,
		},
		{
			name: "failure - claim returns error",
			notif: &entity.Notification{
				NotificationID: "notif-004",
				Title:          "Test Title",
				Body:           "Test Body",
				Status:         constant.Pending,
				Channels: datatypes.NewJSONSlice([]entity.NotificationChannelDestination{
					{Channel: constant.Email, Destination: "test@example.com"},
				}),
			},
			setupMock: func(repo *MockNotificationRepo, provider *MockNotificationProvider) {
				// Claim fails with unexpected error
				repo.EXPECT().
					ClaimNotificationForProcessing(gomock.Any(), "notif-004", gomock.Any()).
					Return(nil, stderrors.New("database error"))
			},
			wantErr: true,
		},
		{
			name: "success - single channel notification",
			notif: &entity.Notification{
				NotificationID: "notif-005",
				Title:          "Single Channel Title",
				Body:           "Single Channel Body",
				Status:         constant.Pending,
				Channels: datatypes.NewJSONSlice([]entity.NotificationChannelDestination{
					{Channel: constant.Email, Destination: "test@example.com"},
				}),
				Results: datatypes.NewJSONSlice([]entity.NotificationStatusPerChannel{}),
			},
			setupMock: func(repo *MockNotificationRepo, provider *MockNotificationProvider) {
				// Claim notification
				repo.EXPECT().
					ClaimNotificationForProcessing(gomock.Any(), "notif-005", gomock.Any()).
					Return(&entity.Notification{
						NotificationID: "notif-005",
						Title:          "Single Channel Title",
						Body:           "Single Channel Body",
						Status:         constant.Processing,
						Channels: datatypes.NewJSONSlice([]entity.NotificationChannelDestination{
							{Channel: constant.Email, Destination: "test@example.com"},
						}),
						Results: datatypes.NewJSONSlice([]entity.NotificationStatusPerChannel{}),
					}, nil)

				// Provider sends - succeeds
				provider.EXPECT().
					Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)

				// Update channel result
				repo.EXPECT().
					UpdateChannelResult(gomock.Any(), "notif-005", gomock.Any()).
					Return(nil)

				// Get notification for final status
				repo.EXPECT().
					GetByNotificationID(gomock.Any(), "notif-005").
					Return(&entity.Notification{
						NotificationID: "notif-005",
						Status:         constant.Processing,
						Results: datatypes.NewJSONSlice([]entity.NotificationStatusPerChannel{
							{Channel: constant.Email, Status: constant.Sent},
						}),
					}, nil)

				// Release notification
				repo.EXPECT().
					ReleaseNotificationFromProcessing(gomock.Any(), "notif-005", gomock.Any(), constant.Sent).
					Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := NewMockNotificationRepo(ctrl)
			mockProvider := NewMockNotificationProvider(ctrl)
			mockProvider.EXPECT().
				GetChannel().
				Return(constant.Email).
				AnyTimes()

			tt.setupMock(mockRepo, mockProvider)

			logger := log.NewStdLogger(io.Discard)
			registry := NewProviderRegistry()
			registry.Register(mockProvider)

			worker := NewNotificationWorker(mockRepo, registry, logger)

			err := worker.processNotification(context.Background(), tt.notif)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNotificationWorker_sendToChannel(t *testing.T) {
	tests := []struct {
		name      string
		notif     *entity.Notification
		channel   constant.NotificationChannel
		dest      string
		setupMock func(*MockNotificationRepo, *MockNotificationProvider)
	}{
		{
			name: "success - send email",
			notif: &entity.Notification{
				NotificationID: "notif-001",
				Title:          "Test",
				Body:           "Body",
			},
			channel: constant.Email,
			dest:    "test@example.com",
			setupMock: func(repo *MockNotificationRepo, provider *MockNotificationProvider) {
				provider.EXPECT().
					GetChannel().
					Return(constant.Email)

				provider.EXPECT().
					Send(gomock.Any(), "test@example.com", "Test", "Body", gomock.Any()).
					Return(nil)

				repo.EXPECT().
					UpdateChannelResult(gomock.Any(), "notif-001", gomock.Any()).
					Do(func(ctx context.Context, notifID string, result entity.NotificationStatusPerChannel) {
						assert.Equal(t, constant.Email, result.Channel)
						assert.Equal(t, constant.Sent, result.Status)
						assert.Empty(t, result.FailureReason)
					}).
					Return(nil)
			},
		},
		{
			name: "failure - provider send fails",
			notif: &entity.Notification{
				NotificationID: "notif-002",
				Title:          "Test",
				Body:           "Body",
			},
			channel: constant.Email,
			dest:    "test@example.com",
			setupMock: func(repo *MockNotificationRepo, provider *MockNotificationProvider) {
				provider.EXPECT().
					GetChannel().
					Return(constant.Email)

				provider.EXPECT().
					Send(gomock.Any(), "test@example.com", "Test", "Body", gomock.Any()).
					Return(stderrors.New("SMTP connection failed"))

				repo.EXPECT().
					UpdateChannelResult(gomock.Any(), "notif-002", gomock.Any()).
					Do(func(ctx context.Context, notifID string, result entity.NotificationStatusPerChannel) {
						assert.Equal(t, constant.Email, result.Channel)
						assert.Equal(t, constant.Failed, result.Status)
						assert.Contains(t, result.FailureReason, "SMTP connection failed")
					}).
					Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := NewMockNotificationRepo(ctrl)
			mockProvider := NewMockNotificationProvider(ctrl)

			tt.setupMock(mockRepo, mockProvider)

			logger := log.NewStdLogger(io.Discard)
			registry := NewProviderRegistry()
			registry.Register(mockProvider)

			worker := NewNotificationWorker(mockRepo, registry, logger)
			worker.sendToChannel(context.Background(), tt.notif, tt.channel, tt.dest)
		})
	}
}

func TestNotificationWorker_determineFinalStatus(t *testing.T) {
	tests := []struct {
		name           string
		notif          *entity.Notification
		setupMock      func(*MockNotificationRepo)
		expectedStatus constant.NotificationStatus
	}{
		{
			name: "all channels sent - status should be sent",
			notif: &entity.Notification{
				NotificationID: "notif-001",
			},
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					GetByNotificationID(gomock.Any(), "notif-001").
					Return(&entity.Notification{
						NotificationID: "notif-001",
						Results: datatypes.NewJSONSlice([]entity.NotificationStatusPerChannel{
							{Channel: constant.Email, Status: constant.Sent},
							{Channel: constant.Push, Status: constant.Sent},
						}),
					}, nil)
			},
			expectedStatus: constant.Sent,
		},
		{
			name: "at least one channel sent - status should be sent",
			notif: &entity.Notification{
				NotificationID: "notif-002",
			},
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					GetByNotificationID(gomock.Any(), "notif-002").
					Return(&entity.Notification{
						NotificationID: "notif-002",
						Results: datatypes.NewJSONSlice([]entity.NotificationStatusPerChannel{
							{Channel: constant.Email, Status: constant.Sent},
							{Channel: constant.Push, Status: constant.Failed},
						}),
					}, nil)
			},
			expectedStatus: constant.Sent,
		},
		{
			name: "all channels failed - status should be failed",
			notif: &entity.Notification{
				NotificationID: "notif-003",
			},
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					GetByNotificationID(gomock.Any(), "notif-003").
					Return(&entity.Notification{
						NotificationID: "notif-003",
						Results: datatypes.NewJSONSlice([]entity.NotificationStatusPerChannel{
							{Channel: constant.Email, Status: constant.Failed},
							{Channel: constant.Push, Status: constant.Failed},
						}),
					}, nil)
			},
			expectedStatus: constant.Failed,
		},
		{
			name: "no results - status should be failed",
			notif: &entity.Notification{
				NotificationID: "notif-004",
			},
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					GetByNotificationID(gomock.Any(), "notif-004").
					Return(&entity.Notification{
						NotificationID: "notif-004",
						Results:        datatypes.NewJSONSlice([]entity.NotificationStatusPerChannel{}),
					}, nil)
			},
			expectedStatus: constant.Failed,
		},
		{
			name: "get notification fails - status should be failed",
			notif: &entity.Notification{
				NotificationID: "notif-005",
			},
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					GetByNotificationID(gomock.Any(), "notif-005").
					Return(nil, stderrors.New("database error"))
			},
			expectedStatus: constant.Failed,
		},
		{
			name: "mixed status with pending - status should be pending",
			notif: &entity.Notification{
				NotificationID: "notif-006",
			},
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					GetByNotificationID(gomock.Any(), "notif-006").
					Return(&entity.Notification{
						NotificationID: "notif-006",
						Results: datatypes.NewJSONSlice([]entity.NotificationStatusPerChannel{
							{Channel: constant.Email, Status: constant.Failed},
							{Channel: constant.Push, Status: constant.Pending},
						}),
					}, nil)
			},
			expectedStatus: constant.Pending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := NewMockNotificationRepo(ctrl)
			tt.setupMock(mockRepo)

			logger := log.NewStdLogger(io.Discard)
			registry := NewProviderRegistry()

			worker := NewNotificationWorker(mockRepo, registry, logger)
			status := worker.determineFinalStatus(tt.notif)

			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestNotificationWorker_updateChannelStatus(t *testing.T) {
	tests := []struct {
		name          string
		notificationID string
		channel       constant.NotificationChannel
		status        constant.NotificationStatus
		failureReason string
		setupMock     func(*MockNotificationRepo)
	}{
		{
			name:           "success - update sent status",
			notificationID: "notif-001",
			channel:        constant.Email,
			status:         constant.Sent,
			failureReason:  "",
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					UpdateChannelResult(gomock.Any(), "notif-001", gomock.Any()).
					Do(func(ctx context.Context, notifID string, result entity.NotificationStatusPerChannel) {
						assert.Equal(t, constant.Email, result.Channel)
						assert.Equal(t, constant.Sent, result.Status)
						assert.Empty(t, result.FailureReason)
						assert.NotNil(t, result.CompletedAt)
					}).
					Return(nil)
			},
		},
		{
			name:           "success - update failed status with reason",
			notificationID: "notif-002",
			channel:        constant.Push,
			status:         constant.Failed,
			failureReason:  "Invalid device token",
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					UpdateChannelResult(gomock.Any(), "notif-002", gomock.Any()).
					Do(func(ctx context.Context, notifID string, result entity.NotificationStatusPerChannel) {
						assert.Equal(t, constant.Push, result.Channel)
						assert.Equal(t, constant.Failed, result.Status)
						assert.Equal(t, "Invalid device token", result.FailureReason)
						assert.NotNil(t, result.CompletedAt)
					}).
					Return(nil)
			},
		},
		{
			name:           "failure - update fails",
			notificationID: "notif-003",
			channel:        constant.Email,
			status:         constant.Sent,
			failureReason:  "",
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					UpdateChannelResult(gomock.Any(), "notif-003", gomock.Any()).
					Return(stderrors.New("database error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := NewMockNotificationRepo(ctrl)
			tt.setupMock(mockRepo)

			logger := log.NewStdLogger(io.Discard)
			registry := NewProviderRegistry()

			worker := NewNotificationWorker(mockRepo, registry, logger)
			worker.updateChannelStatus(
				context.Background(),
				tt.notificationID,
				tt.channel,
				tt.status,
				tt.failureReason,
			)
		})
	}
}
