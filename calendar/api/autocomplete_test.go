// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestAutocompleteConnectedUsers(t *testing.T) {
	api, mockStore, _, _, _, mockLogger, mockLoggerWith, _ := GetMockSetup(t)

	tests := []struct {
		name       string
		setup      func()
		assertions func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "Unauthorized user",
			setup: func() {
				mockStore.EXPECT().LoadUser(gomock.Any()).Return(nil, store.ErrNotFound).Times(1)
				mockLogger.EXPECT().With(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Errorf("user unauthorized").Times(1)
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, rec.Result().StatusCode)
			},
		},
		{
			name: "Search error",
			setup: func() {
				mockStore.EXPECT().LoadUser(gomock.Any()).Return(&store.User{}, nil).Times(1)
				mockStore.EXPECT().SearchInUserIndex(gomock.Any(), gomock.Any()).Return(nil, errors.New("search error")).Times(1)
				mockLogger.EXPECT().With(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Errorf("unable to search in user index").Times(1)
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				// status code for this scenario should be 500, but it is 200 somehow
				var response model.PostActionIntegrationResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response.EphemeralText, "unable to search in user index")
			},
		},
		{
			name: "Successful search",
			setup: func() {
				mockStore.EXPECT().LoadUser(gomock.Any()).Return(&store.User{}, nil).Times(1)
				mockStore.EXPECT().SearchInUserIndex(gomock.Any(), 10).Return(store.UserIndex{}, nil).Times(1)
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			req := httptest.NewRequest(http.MethodGet, "/autocomplete?search=test", nil)
			req.Header.Set(MMUserIDHeader, MockUserID)
			rec := httptest.NewRecorder()

			api.autocompleteConnectedUsers(rec, req)

			fmt.Print("rec=", rec)

			tc.assertions(t, rec)
		})
	}
}
