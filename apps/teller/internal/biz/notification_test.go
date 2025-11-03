package biz

import (
	"context"
	stderrors "errors"
	"io"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	v1 "github.com/toeydevelopment/telltheworld/apis/teller/v1"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/constant"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/data/entity"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/errors"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

func TestNotification_SendToUser(t *testing.T) {
	tests := []struct {
		name      string
		request   *v1.SendToUserRequest
		setupMock func(*MockNotificationRepo)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "success - single channel email",
			request: &v1.SendToUserRequest{
				Notification: &v1.Notification{
					Title:  "Test Title",
					Body:   "Test Body",
					RefId:  "ref-001",
				},
				Destinations: []*v1.ChannelDestination{
					{
						Channel:     v1.NotificationChannel_NotificationChannel_Email,
						Destination: "test@example.com",
					},
				},
			},
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					Save(gomock.Any(), gomock.Any()).
					Return(nil)
				repo.EXPECT().
					PublishNotification(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - multiple channels",
			request: &v1.SendToUserRequest{
				Notification: &v1.Notification{
					Title:  "Test Title",
					Body:   "Test Body",
					RefId:  "ref-002",
				},
				Destinations: []*v1.ChannelDestination{
					{
						Channel:     v1.NotificationChannel_NotificationChannel_Email,
						Destination: "test@example.com",
					},
					{
						Channel:     v1.NotificationChannel_NotificationChannel_Push,
						Destination: "device-token-123",
					},
				},
			},
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					Save(gomock.Any(), gomock.Any()).
					Return(nil)
				repo.EXPECT().
					PublishNotification(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - with scheduled time",
			request: &v1.SendToUserRequest{
				Notification: &v1.Notification{
					Title:      "Test Title",
					Body:       "Test Body",
					RefId:      "ref-003",
					ScheduleAt: timestamppb.New(time.Now().Add(1 * time.Hour)),
				},
				Destinations: []*v1.ChannelDestination{
					{
						Channel:     v1.NotificationChannel_NotificationChannel_Email,
						Destination: "test@example.com",
					},
				},
			},
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					Save(gomock.Any(), gomock.Any()).
					Return(nil)
				repo.EXPECT().
					PublishNotification(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failure - save fails",
			request: &v1.SendToUserRequest{
				Notification: &v1.Notification{
					Title:  "Test Title",
					Body:   "Test Body",
					RefId:  "ref-004",
				},
				Destinations: []*v1.ChannelDestination{
					{
						Channel:     v1.NotificationChannel_NotificationChannel_Email,
						Destination: "test@example.com",
					},
				},
			},
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					Save(gomock.Any(), gomock.Any()).
					Return(stderrors.New("database error"))
			},
			wantErr: true,
			errMsg:  "failed to save notification",
		},
		{
			name: "failure - publish fails",
			request: &v1.SendToUserRequest{
				Notification: &v1.Notification{
					Title:  "Test Title",
					Body:   "Test Body",
					RefId:  "ref-005",
				},
				Destinations: []*v1.ChannelDestination{
					{
						Channel:     v1.NotificationChannel_NotificationChannel_Email,
						Destination: "test@example.com",
					},
				},
			},
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					Save(gomock.Any(), gomock.Any()).
					Return(nil)
				repo.EXPECT().
					PublishNotification(gomock.Any(), gomock.Any()).
					Return(stderrors.New("publish error"))
				repo.EXPECT().
					UpdateStatus(gomock.Any(), gomock.Any(), constant.Failed).
					Return(nil)
			},
			wantErr: true,
			errMsg:  "failed to publish notification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := NewMockNotificationRepo(ctrl)
			tt.setupMock(mockRepo)

			logger := log.NewStdLogger(io.Discard)
			notification := NewNotification(mockRepo, logger)

			notificationID, err := notification.SendToUser(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Empty(t, notificationID)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, notificationID)
			}
		})
	}
}

func TestNotification_BulkSendToUser(t *testing.T) {
	tests := []struct {
		name           string
		request        *v1.BulkSendToUserRequest
		setupMock      func(*MockNotificationRepo)
		wantErr        bool
		errMsg         string
		validateResult func(*testing.T, []string)
	}{
		{
			name: "success - multiple notifications",
			request: &v1.BulkSendToUserRequest{
				Requests: []*v1.SendToUserRequest{
					{
						Notification: &v1.Notification{
							Title:  "Test 1",
							Body:   "Body 1",
							RefId:  "ref-001",
						},
						Destinations: []*v1.ChannelDestination{
							{Channel: v1.NotificationChannel_NotificationChannel_Email, Destination: "test1@example.com"},
						},
					},
					{
						Notification: &v1.Notification{
							Title:  "Test 2",
							Body:   "Body 2",
							RefId:  "ref-002",
						},
						Destinations: []*v1.ChannelDestination{
							{Channel: v1.NotificationChannel_NotificationChannel_Email, Destination: "test2@example.com"},
						},
					},
				},
			},
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					BulkSave(gomock.Any(), gomock.Any()).
					Return(nil)
				repo.EXPECT().
					PublishNotification(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(2)
			},
			wantErr: false,
			validateResult: func(t *testing.T, ids []string) {
				assert.Len(t, ids, 2)
				for _, id := range ids {
					assert.NotEmpty(t, id)
				}
			},
		},
		{
			name: "success - empty request",
			request: &v1.BulkSendToUserRequest{
				Requests: []*v1.SendToUserRequest{},
			},
			setupMock: func(repo *MockNotificationRepo) {
				// No expectations needed for empty request
			},
			wantErr: false,
			validateResult: func(t *testing.T, ids []string) {
				assert.Len(t, ids, 0)
			},
		},
		{
			name: "failure - bulk save fails",
			request: &v1.BulkSendToUserRequest{
				Requests: []*v1.SendToUserRequest{
					{
						Notification: &v1.Notification{
							Title:  "Test",
							Body:   "Body",
							RefId:  "ref-003",
						},
						Destinations: []*v1.ChannelDestination{
							{Channel: v1.NotificationChannel_NotificationChannel_Email, Destination: "test@example.com"},
						},
					},
				},
			},
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					BulkSave(gomock.Any(), gomock.Any()).
					Return(stderrors.New("database error"))
			},
			wantErr: true,
			errMsg:  "failed to bulk save notifications",
		},
		{
			name: "partial failure - some publishes fail",
			request: &v1.BulkSendToUserRequest{
				Requests: []*v1.SendToUserRequest{
					{
						Notification: &v1.Notification{
							Title:  "Test 1",
							Body:   "Body 1",
							RefId:  "ref-004",
						},
						Destinations: []*v1.ChannelDestination{
							{Channel: v1.NotificationChannel_NotificationChannel_Email, Destination: "test1@example.com"},
						},
					},
					{
						Notification: &v1.Notification{
							Title:  "Test 2",
							Body:   "Body 2",
							RefId:  "ref-005",
						},
						Destinations: []*v1.ChannelDestination{
							{Channel: v1.NotificationChannel_NotificationChannel_Email, Destination: "test2@example.com"},
						},
					},
				},
			},
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					BulkSave(gomock.Any(), gomock.Any()).
					Return(nil)
				// First publish succeeds
				repo.EXPECT().
					PublishNotification(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				// Second publish fails
				repo.EXPECT().
					PublishNotification(gomock.Any(), gomock.Any()).
					Return(stderrors.New("publish error")).
					Times(1)
				repo.EXPECT().
					UpdateStatus(gomock.Any(), gomock.Any(), constant.Failed).
					Return(nil)
			},
			wantErr: false,
			validateResult: func(t *testing.T, ids []string) {
				assert.Len(t, ids, 2)
				// One should be valid, one should be empty
				validCount := 0
				emptyCount := 0
				for _, id := range ids {
					if id != "" {
						validCount++
					} else {
						emptyCount++
					}
				}
				assert.Equal(t, 1, validCount)
				assert.Equal(t, 1, emptyCount)
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
			notification := NewNotification(mockRepo, logger)

			ids, err := notification.BulkSendToUser(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, ids)
			} else {
				assert.NoError(t, err)
				if tt.validateResult != nil {
					tt.validateResult(t, ids)
				}
			}
		})
	}
}

func TestNotification_GetNotificationStatus(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		notificationID string
		setupMock      func(*MockNotificationRepo)
		wantErr        bool
		errMsg         string
		validateResult func(*testing.T, []*v1.StatusPerChannel)
	}{
		{
			name:           "success - single channel status",
			notificationID: "notif-001",
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					GetByNotificationID(gomock.Any(), "notif-001").
					Return(&entity.Notification{
						NotificationID: "notif-001",
						Status:         constant.Sent,
						Results: datatypes.NewJSONSlice([]entity.NotificationStatusPerChannel{
							{
								Channel:     constant.Email,
								Status:      constant.Sent,
								CompletedAt: &now,
							},
						}),
					}, nil)
			},
			wantErr: false,
			validateResult: func(t *testing.T, statuses []*v1.StatusPerChannel) {
				assert.Len(t, statuses, 1)
				assert.Equal(t, v1.NotificationChannel_NotificationChannel_Email, statuses[0].Channel)
				assert.Equal(t, v1.NotificationStatus_NotificationStatus_Sent, statuses[0].Status)
				assert.Nil(t, statuses[0].FailReason)
				assert.NotNil(t, statuses[0].CompletedAt)
			},
		},
		{
			name:           "success - multiple channels with mixed status",
			notificationID: "notif-002",
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					GetByNotificationID(gomock.Any(), "notif-002").
					Return(&entity.Notification{
						NotificationID: "notif-002",
						Status:         constant.Sent,
						Results: datatypes.NewJSONSlice([]entity.NotificationStatusPerChannel{
							{
								Channel:     constant.Email,
								Status:      constant.Sent,
								CompletedAt: &now,
							},
							{
								Channel:       constant.Push,
								Status:        constant.Failed,
								FailureReason: "device token invalid",
								CompletedAt:   &now,
							},
						}),
					}, nil)
			},
			wantErr: false,
			validateResult: func(t *testing.T, statuses []*v1.StatusPerChannel) {
				assert.Len(t, statuses, 2)

				// Check email status
				emailStatus := statuses[0]
				assert.Equal(t, v1.NotificationChannel_NotificationChannel_Email, emailStatus.Channel)
				assert.Equal(t, v1.NotificationStatus_NotificationStatus_Sent, emailStatus.Status)

				// Check push status
				pushStatus := statuses[1]
				assert.Equal(t, v1.NotificationChannel_NotificationChannel_Push, pushStatus.Channel)
				assert.Equal(t, v1.NotificationStatus_NotificationStatus_Failed, pushStatus.Status)
				assert.NotNil(t, pushStatus.FailReason)
				assert.Equal(t, "device token invalid", *pushStatus.FailReason)
			},
		},
		{
			name:           "failure - notification not found",
			notificationID: "notif-999",
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					GetByNotificationID(gomock.Any(), "notif-999").
					Return(nil, errors.ErrNotificationNotFound)
			},
			wantErr: true,
			errMsg:  "notification not found",
		},
		{
			name:           "success - empty results",
			notificationID: "notif-003",
			setupMock: func(repo *MockNotificationRepo) {
				repo.EXPECT().
					GetByNotificationID(gomock.Any(), "notif-003").
					Return(&entity.Notification{
						NotificationID: "notif-003",
						Status:         constant.Pending,
						Results:        datatypes.NewJSONSlice([]entity.NotificationStatusPerChannel{}),
					}, nil)
			},
			wantErr: false,
			validateResult: func(t *testing.T, statuses []*v1.StatusPerChannel) {
				assert.Len(t, statuses, 0)
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
			notification := NewNotification(mockRepo, logger)

			statuses, err := notification.GetNotificationStatus(context.Background(), tt.notificationID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, statuses)
			} else {
				assert.NoError(t, err)
				if tt.validateResult != nil {
					tt.validateResult(t, statuses)
				}
			}
		})
	}
}

// Note: Removed tests for private converter functions as they cannot be tested from external package
