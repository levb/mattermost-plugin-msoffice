// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestToRemoteEvent(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	assert.NoError(t, err)

	tests := []struct {
		name       string
		payload    createEventPayload
		assertions func(t *testing.T, event *remote.Event, err error)
	}{
		{
			name:    "Invalid start time format",
			payload: GetMockCreateEventPayload(false, nil, "2024-10-18", "invalid_time", "", "", "", "", ""),
			assertions: func(t *testing.T, event *remote.Event, err error) {
				assert.Error(t, err)
				assert.Nil(t, event)
			},
		},
		{
			name:    "Invalid end time format",
			payload: GetMockCreateEventPayload(false, nil, "2024-10-18", "10:00", "invalid_time", "", "", "", ""),
			assertions: func(t *testing.T, event *remote.Event, err error) {
				assert.Error(t, err)
				assert.Nil(t, event)
			},
		},
		{
			name:    "Invalid date format",
			payload: GetMockCreateEventPayload(false, nil, "18-10-2024", "", "", "", "", "", ""),
			assertions: func(t *testing.T, event *remote.Event, err error) {
				assert.Error(t, err)
				assert.Nil(t, event)
			},
		},
		{
			name:    "Valid all-day event",
			payload: GetMockCreateEventPayload(true, nil, "2024-10-18", "10:00", "12:00", "Meeting with team", "Conference Room", "Discuss the quarterly results.", ""),
			assertions: func(t *testing.T, event *remote.Event, err error) {
				assert.NoError(t, err)
				assert.True(t, event.IsAllDay)
			},
		},
		{
			name:    "Valid event with specific start and end time",
			payload: GetMockCreateEventPayload(false, nil, "2024-10-18", "10:00", "12:00", "Discuss the quarterly results.", "Meeting with team", "Conference Room", ""),
			assertions: func(t *testing.T, event *remote.Event, err error) {
				expectedEvent := &remote.Event{
					IsAllDay: false,
					Start: &remote.DateTime{
						DateTime: "2024-10-18T10:00:00",
						TimeZone: "America/New_York",
					},
					End: &remote.DateTime{
						DateTime: "2024-10-18T12:00:00",
						TimeZone: "America/New_York",
					},
					Subject: "Meeting with team",
					Location: &remote.Location{
						DisplayName: "Conference Room",
					},
					Body: &remote.ItemBody{
						Content:     "Discuss the quarterly results.",
						ContentType: "text/plain",
					},
				}
				assert.Equal(t, expectedEvent, event)
				assert.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := tt.payload.ToRemoteEvent(loc)

			tt.assertions(t, event, err)
		})
	}
}

func TestIsValid(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	assert.NoError(t, err)

	tests := []struct {
		name       string
		payload    createEventPayload
		assertions func(t *testing.T, err error)
	}{
		{
			name:    "Missing subject",
			payload: GetMockCreateEventPayload(false, nil, "2024-10-18", "10:00", "12:00", "mockDescription", "", "mockLocation", ""),
			assertions: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "subject must not be empty")
			},
		},
		{
			name:    "Missing date",
			payload: GetMockCreateEventPayload(false, nil, "", "10:00", "12:00", "mockDescription", "mockSubject", "mockLocation", ""),
			assertions: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "date must not be empty")
			},
		},
		{
			name:    "Invalid date format",
			payload: GetMockCreateEventPayload(false, nil, "18-10-2024", "10:00", "12:00", "mockDescription", "mockSubject", "mockLocation", ""),
			assertions: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "invalid date")
			},
		},
		{
			name:    "Missing start and end time for non-all-day event",
			payload: GetMockCreateEventPayload(false, nil, "2024-10-18", "", "", "mockDescription", "mockSubject", "mockLocation", ""),
			assertions: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "start time/end time must be set or event should last all day")
			},
		},
		{
			name:    "Invalid start time",
			payload: GetMockCreateEventPayload(false, nil, "2024-10-18", "invalidStartTime", "12:00", "mockDescription", "mockSubject", "mockLocation", ""),
			assertions: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "please use a valid start time")
			},
		},
		{
			name:    "Start time in the past",
			payload: GetMockCreateEventPayload(false, nil, "2022-10-18", "10:20", "12:00", "mockDescription", "mockSubject", "mockLocation", ""),
			assertions: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "please select a start date and time that is not prior to the current time")
			},
		},
		{
			name:    "Invalid end time",
			payload: GetMockCreateEventPayload(false, nil, time.Now().Format("2006-01-02"), time.Now().Add(1*time.Hour).Format("15:04"), "invalidEndTime", "mockDescription", "mockSubject", "mockLocation", ""),
			assertions: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "please use a valid end time")
			},
		},
		{
			name:    "End time before start time",
			payload: GetMockCreateEventPayload(false, nil, time.Now().Format("2006-01-02"), time.Now().Add(2*time.Hour).Format("15:04"), time.Now().Add(1*time.Hour).Format("15:04"), "mockDescription", "mockSubject", "mockLocation", ""),
			assertions: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "end date cannot be earlier than start date")
			},
		},
		{
			name:    "Valid event",
			payload: GetMockCreateEventPayload(false, nil, time.Now().Format("2006-01-02"), time.Now().Add(1*time.Hour).Format("15:04"), time.Now().Add(2*time.Hour).Format("15:04"), "mockDescription", "mockSubject", "mockLocation", ""),
			assertions: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payload.IsValid(loc)
			tt.assertions(t, err)
		})
	}
}

func TestCreateEvent(t *testing.T) {
	api, mockStore, mockPoster, mockRemote, mockPluginAPI, mockLogger, mockLoggerWith, mockRemoteClient := GetMockSetup(t)

	tests := []struct {
		name       string
		setup      func(*http.Request)
		assertions func(*httptest.ResponseRecorder)
	}{
		{
			name: "Missing Mattermost User ID",
			setup: func(req *http.Request) {
				req.Header.Del(MMUserIDHeader)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						"value": true,
					},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				mockLogger.EXPECT().Errorf("createEvent, unauthorized user").Times(1)
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				responseBody, readErr := io.ReadAll(rec.Body)
				assert.NoError(t, readErr)
				assert.Contains(t, string(responseBody), "unauthorized")
				assert.Equal(t, http.StatusUnauthorized, rec.Result().StatusCode)
			},
		},
		{
			name: "Error loading the user",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						"value": true,
					},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				mockStore.EXPECT().LoadUser(MockUserID).Return(nil, errors.New("internal error")).Times(1)
				mockLogger.EXPECT().With(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Errorf("createEvent, error occurred while loading user from store").Times(1)
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				responseBody, readErr := io.ReadAll(rec.Body)
				assert.NoError(t, readErr)
				assert.Contains(t, string(responseBody), "internal error")
				assert.Equal(t, http.StatusInternalServerError, rec.Result().StatusCode)
			},
		},
		{
			name: "User not found",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						"value": true,
					},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				mockStore.EXPECT().LoadUser(MockUserID).Return(nil, store.ErrNotFound).Times(1)
				mockLogger.EXPECT().With(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Errorf("createEvent, user not found in store").Times(1)
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				responseBody, readErr := io.ReadAll(rec.Body)
				assert.NoError(t, readErr)
				assert.Contains(t, string(responseBody), "unauthorized")
				assert.Equal(t, http.StatusUnauthorized, rec.Result().StatusCode)
			},
		},
		{
			name: "Error decoding the event payload",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				malformedJSON := `{"all_day": true, "attendees": ["user1", "user2"], "date": "2024-10-17", "start_time": "10:00AM", "end_time": }`
				req.Body = io.NopCloser(bytes.NewBufferString(malformedJSON))
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{}, nil).Times(1)
				mockLogger.EXPECT().With(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Errorf("createEvent, error occurred while decoding event payload").Times(1)
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				responseBody, readErr := io.ReadAll(rec.Body)
				assert.NoError(t, readErr)
				assert.Contains(t, string(responseBody), "invalid character")
				assert.Equal(t, http.StatusBadRequest, rec.Result().StatusCode)
			},
		},
		{
			name: "User doesn't have permission to link event to the channel",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				req.Body = io.NopCloser(bytes.NewBufferString(ValidRequestBodyJSON))
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{MattermostUserID: MockUserID}, nil).Times(1)
				mockPluginAPI.EXPECT().CanLinkEventToChannel(MockChannelID, MockUserID).Return(false).Times(1)
				mockLogger.EXPECT().With(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Errorf("createEvent, user don't have permission to link events in the selected channel").Times(1)
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Result().StatusCode)
				responseBody, _ := io.ReadAll(rec.Body)
				assert.Contains(t, string(responseBody), "you don't have permission to link events in the selected channel")
			},
		},
		{
			name: "Error getting the mailbox settings",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				req.Body = io.NopCloser(bytes.NewBufferString(ValidRequestBodyJSON))
				mockOAauthToken := oauth2.Token{}
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{MattermostUserID: MockUserID, OAuth2Token: &mockOAauthToken, Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(1)
				mockPluginAPI.EXPECT().CanLinkEventToChannel(MockChannelID, MockUserID).Return(true).Times(1)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), &mockOAauthToken, gomock.Any(), gomock.Any(), gomock.Any()).Return(mockRemoteClient).Times(1)
				mockRemoteClient.EXPECT().GetMailboxSettings(MockRemoteUserID).Return(nil, errors.New("error getting mailbox settings")).Times(1)
				mockLogger.EXPECT().With(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Errorf("createEvent, error occurred while getting mailbox settings for user").Times(1)
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				responseBody, _ := io.ReadAll(rec.Body)
				assert.Contains(t, string(responseBody), "error getting mailbox settings")
				assert.Equal(t, http.StatusInternalServerError, rec.Result().StatusCode)
			},
		},
		{
			name: "Error loading mailbox timezone location",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				req.Body = io.NopCloser(bytes.NewBufferString(ValidRequestBodyJSON))
				mockOAauthToken := oauth2.Token{}
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{MattermostUserID: MockUserID, OAuth2Token: &mockOAauthToken, Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(1)
				mockPluginAPI.EXPECT().CanLinkEventToChannel(MockChannelID, MockUserID).Return(true).Times(1)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), &mockOAauthToken, gomock.Any(), gomock.Any(), gomock.Any()).Return(mockRemoteClient).Times(1)
				mockRemoteClient.EXPECT().GetMailboxSettings(MockRemoteUserID).Return(&remote.MailboxSettings{TimeZone: "Invalid/TimeZone"}, nil).Times(1)
				mockLogger.EXPECT().With(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Errorf("createEvent, error occurred while loading mailbox timezone location").Times(1)
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, rec.Result().StatusCode)
				responseBody, _ := io.ReadAll(rec.Body)
				assert.Contains(t, string(responseBody), "unknown time zone")
			},
		},
		{
			name: "Invalid payload",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				req.Body = io.NopCloser(bytes.NewBufferString(ValidRequestBodyJSON))
				mockOAauthToken := oauth2.Token{}
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{MattermostUserID: MockUserID, OAuth2Token: &mockOAauthToken, Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(1)
				mockPluginAPI.EXPECT().CanLinkEventToChannel(MockChannelID, MockUserID).Return(true).Times(1)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), &mockOAauthToken, gomock.Any(), gomock.Any(), gomock.Any()).Return(mockRemoteClient).Times(1)
				mockRemoteClient.EXPECT().GetMailboxSettings(MockRemoteUserID).Return(&remote.MailboxSettings{TimeZone: "UTC"}, nil).Times(1)
				mockLogger.EXPECT().Errorf("createEvent, invalid payload").Times(1)
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Result().StatusCode)
				responseBody, _ := io.ReadAll(rec.Body)
				assert.Contains(t, string(responseBody), "Invalid request.")
			},
		},
		{
			name: "Error creating event",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				validJSON := GetCurrentTimeRequestBodyJSON(MockChannelID)
				req.Body = io.NopCloser(bytes.NewBufferString(validJSON))
				mockOAauthToken := oauth2.Token{}
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{MattermostUserID: MockUserID, OAuth2Token: &mockOAauthToken, Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(1)
				mockPluginAPI.EXPECT().CanLinkEventToChannel(MockChannelID, MockUserID).Return(true).Times(1)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), &mockOAauthToken, gomock.Any(), gomock.Any(), gomock.Any()).Return(mockRemoteClient).Times(1)
				mockRemoteClient.EXPECT().GetMailboxSettings(MockRemoteUserID).Return(&remote.MailboxSettings{TimeZone: "UTC"}, nil).Times(1)
				mockRemoteClient.EXPECT().CreateEvent(MockRemoteUserID, gomock.Any()).Return(nil, errors.New("failed to create event")).Times(1)
				mockLogger.EXPECT().With(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Errorf("createEvent, error occurred while creating event", gomock.Any()).Times(1)
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, rec.Result().StatusCode)
				responseBody, _ := io.ReadAll(rec.Body)
				assert.Contains(t, string(responseBody), "failed to create event")
			},
		},
		{
			name: "Error storing the linked user event",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				validJSON := GetCurrentTimeRequestBodyJSON(MockChannelID)
				req.Body = io.NopCloser(bytes.NewBufferString(validJSON))
				mockOAauthToken := oauth2.Token{}
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{MattermostUserID: MockUserID, OAuth2Token: &mockOAauthToken, Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(1)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), &mockOAauthToken, gomock.Any(), gomock.Any(), gomock.Any()).Return(mockRemoteClient).Times(1)
				mockPluginAPI.EXPECT().CanLinkEventToChannel(MockChannelID, MockUserID).Return(true).Times(1)
				mockRemoteClient.EXPECT().GetMailboxSettings(MockRemoteUserID).Return(&remote.MailboxSettings{TimeZone: "UTC"}, nil).Times(1)
				mockEvent := GetMockRemoteEvent()
				mockRemoteClient.EXPECT().CreateEvent(MockRemoteUserID, gomock.Any()).Return(mockEvent, nil).Times(1)
				mockStore.EXPECT().StoreUserLinkedEvent(MockUserID, gomock.Any(), MockChannelID).Return(errors.New("error storing the user linked event")).Times(1)
				mockPoster.EXPECT().DM(MockUserID, gomock.Any(), gomock.Any()).Times(1)
				mockLogger.EXPECT().With(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Errorf("createEvent, error occurred while storing user linked event").Times(1)
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, rec.Result().StatusCode)
				responseBody, _ := io.ReadAll(rec.Body)
				assert.Contains(t, string(responseBody), "error storing the user linked event")
			},
		},
		{
			name: "Error linking event to channel",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				validJSON := GetCurrentTimeRequestBodyJSON(MockChannelID)
				req.Body = io.NopCloser(bytes.NewBufferString(validJSON))
				mockOAauthToken := oauth2.Token{}
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{MattermostUserID: MockUserID, OAuth2Token: &mockOAauthToken, Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(1)
				mockPluginAPI.EXPECT().CanLinkEventToChannel(MockChannelID, MockUserID).Return(true).Times(1)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), &mockOAauthToken, gomock.Any(), gomock.Any(), gomock.Any()).Return(mockRemoteClient).Times(1)
				mockRemoteClient.EXPECT().GetMailboxSettings(MockRemoteUserID).Return(&remote.MailboxSettings{TimeZone: "UTC"}, nil).Times(1)
				mockEvent := GetMockRemoteEvent()
				mockRemoteClient.EXPECT().CreateEvent(MockRemoteUserID, gomock.Any()).Return(mockEvent, nil).Times(1)
				mockStore.EXPECT().StoreUserLinkedEvent(MockUserID, gomock.Any(), MockChannelID).Return(nil).Times(1)
				mockStore.EXPECT().AddLinkedChannelToEvent(gomock.Any(), MockChannelID).Return(errors.New("error linking event to channel")).Times(1)
				mockPoster.EXPECT().DM(MockUserID, gomock.Any(), gomock.Any()).Times(1)
				mockLogger.EXPECT().With(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Errorf("error linking event to channel").Times(1)
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Result().StatusCode)
				responseBody, _ := io.ReadAll(rec.Body)
				assert.Contains(t, string(responseBody), "true")
			},
		},
		{
			name: "Error creating post",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				validJSON := GetCurrentTimeRequestBodyJSON(MockChannelID)
				req.Body = io.NopCloser(bytes.NewBufferString(validJSON))
				mockOAauthToken := oauth2.Token{}
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{MattermostUserID: MockUserID, OAuth2Token: &mockOAauthToken, Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(1)
				mockPluginAPI.EXPECT().CanLinkEventToChannel(MockChannelID, MockUserID).Return(true).Times(1)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), &mockOAauthToken, gomock.Any(), gomock.Any(), gomock.Any()).Return(mockRemoteClient).Times(1)
				mockRemoteClient.EXPECT().GetMailboxSettings(MockRemoteUserID).Return(&remote.MailboxSettings{TimeZone: "UTC"}, nil).Times(1)
				mockEvent := GetMockRemoteEvent()
				mockRemoteClient.EXPECT().CreateEvent(MockRemoteUserID, gomock.Any()).Return(mockEvent, nil).Times(1)
				mockStore.EXPECT().StoreUserLinkedEvent(MockUserID, gomock.Any(), MockChannelID).Return(nil).Times(1)
				mockStore.EXPECT().AddLinkedChannelToEvent(gomock.Any(), MockChannelID).Return(nil).Times(1)
				mockPoster.EXPECT().CreatePost(gomock.Any()).Return(errors.New("error occurred creating post")).Times(1)
				mockLogger.EXPECT().With(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Errorf("error sending post to channel about linked event").Times(1)
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Result().StatusCode)
				responseBody, _ := io.ReadAll(rec.Body)
				assert.Contains(t, string(responseBody), "true")
			},
		},
		{
			name: "Successfully create event with channelID",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				validJSON := GetCurrentTimeRequestBodyJSON(MockChannelID)
				req.Body = io.NopCloser(bytes.NewBufferString(validJSON))
				mockOAauthToken := oauth2.Token{}
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{MattermostUserID: MockUserID, OAuth2Token: &mockOAauthToken, Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(1)
				mockPluginAPI.EXPECT().CanLinkEventToChannel(MockChannelID, MockUserID).Return(true).Times(1)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), &mockOAauthToken, gomock.Any(), gomock.Any(), gomock.Any()).Return(mockRemoteClient).Times(1)
				mockRemoteClient.EXPECT().GetMailboxSettings(MockRemoteUserID).Return(&remote.MailboxSettings{TimeZone: "UTC"}, nil).Times(1)
				mockEvent := GetMockRemoteEvent()
				mockRemoteClient.EXPECT().CreateEvent(MockRemoteUserID, gomock.Any()).Return(mockEvent, nil).Times(1)
				mockStore.EXPECT().StoreUserLinkedEvent(MockUserID, gomock.Any(), MockChannelID).Return(nil).Times(1)
				mockStore.EXPECT().AddLinkedChannelToEvent(gomock.Any(), MockChannelID).Return(nil).Times(1)
				mockPoster.EXPECT().CreatePost(gomock.Any()).Return(nil).Times(1)
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Result().StatusCode)
				responseBody, _ := io.ReadAll(rec.Body)
				assert.Contains(t, string(responseBody), "true")
			},
		},
		{
			name: "Event created successfully without channelID",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				validJSON := GetCurrentTimeRequestBodyJSON("")
				req.Body = io.NopCloser(bytes.NewBufferString(validJSON))
				mockOAauthToken := oauth2.Token{}
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{MattermostUserID: MockUserID, OAuth2Token: &mockOAauthToken, Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(1)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), &mockOAauthToken, gomock.Any(), gomock.Any(), gomock.Any()).Return(mockRemoteClient).Times(1)
				mockRemoteClient.EXPECT().GetMailboxSettings(MockRemoteUserID).Return(&remote.MailboxSettings{TimeZone: "UTC"}, nil).Times(1)
				mockEvent := GetMockRemoteEvent()
				mockRemoteClient.EXPECT().CreateEvent(MockRemoteUserID, gomock.Any()).Return(mockEvent, nil).Times(1)
				mockPoster.EXPECT().DMWithMessageAndAttachments(MockUserID, "Your event was created successfully.", gomock.Any()).Times(1)
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Result().StatusCode)
				responseBody, _ := io.ReadAll(rec.Body)
				assert.Contains(t, string(responseBody), "true")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/create", nil)
			rec := httptest.NewRecorder()

			tc.setup(req)
			api.createEvent(rec, req)

			tc.assertions(rec)
		})
	}
}
