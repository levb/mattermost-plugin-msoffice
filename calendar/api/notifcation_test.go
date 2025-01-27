// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
)

func TestNotification(t *testing.T) {
	api, _, _, mockRemote, _, mockLogger, mockLoggerWith, _ := GetMockSetup(t)
	mockProcessor := &MockNotificationProcessor{}
	api.NotificationProcessor = mockProcessor

	tests := []struct {
		name       string
		setup      func(*MockNotificationProcessor)
		assertions func(*httptest.ResponseRecorder, *MockNotificationProcessor)
	}{
		{
			name: "Error while adding event to notification queue",
			setup: func(mockProcessor *MockNotificationProcessor) {
				mockProcessor.err = errors.New("queue error")
				mockRemote.EXPECT().HandleWebhook(gomock.Any(), gomock.Any()).
					Return([]*remote.Notification{{}}).Times(1)

				mockLogger.EXPECT().With(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Errorf("notification, error occurred while adding webhook event to notification queue").Times(1)
			},
			assertions: func(rec *httptest.ResponseRecorder, mockProcessor *MockNotificationProcessor) {
				assert.Equal(t, http.StatusInternalServerError, rec.Result().StatusCode)
				assert.Equal(t, 0, len(mockProcessor.queue))
			},
		},
		{
			name: "Successful notification processing",
			setup: func(mockProcessor *MockNotificationProcessor) {
				mockProcessor.err = nil
				mockRemote.EXPECT().HandleWebhook(gomock.Any(), gomock.Any()).
					Return([]*remote.Notification{{}, {}}).Times(1)
			},
			assertions: func(rec *httptest.ResponseRecorder, mockProcessor *MockNotificationProcessor) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				assert.Equal(t, 2, len(mockProcessor.queue))
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(mockProcessor)

			req := httptest.NewRequest(http.MethodPost, "/notification", nil)
			rec := httptest.NewRecorder()

			api.notification(rec, req)

			tc.assertions(rec, mockProcessor)
		})
	}
}
