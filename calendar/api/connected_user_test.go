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

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
)

func TestConnectedUserHandler(t *testing.T) {
	api, mockStore, _, _, _, mockLogger, mockLoggerWith, _ := GetMockSetup(t)

	tests := []struct {
		name       string
		setup      func(req *http.Request)
		assertions func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "Missing MattermostUserId in header",
			setup: func(req *http.Request) {
				mockLogger.EXPECT().Errorf("connectedUserHandler, unauthorized user").Times(1)
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, rec.Result().StatusCode)
			},
		},
		{
			name: "Error loading user from store",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				mockStore.EXPECT().LoadUser(MockUserID).Return(nil, errors.New("store error")).Times(1)
				mockLogger.EXPECT().With(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Errorf("connectedUserHandler, error occurred while loading user from store").Times(1)
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, rec.Result().StatusCode)
			},
		},
		{
			name: "User not found in store",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				mockStore.EXPECT().LoadUser(MockUserID).Return(nil, store.ErrNotFound).Times(1)
				mockLogger.EXPECT().With(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Errorf("connectedUserHandler, user not found in store").Times(1)
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, rec.Result().StatusCode)
			},
		},
		{
			name: "User successfully connected",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{}, nil).Times(1)
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				assert.JSONEq(t, `{"is_connected": true}`, rec.Body.String())
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/connected", nil)
			tc.setup(req)
			rec := httptest.NewRecorder()

			api.connectedUserHandler(rec, req)

			tc.assertions(t, rec)
		})
	}
}
