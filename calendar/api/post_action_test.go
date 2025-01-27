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

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestPreprocessAction(t *testing.T) {
	api, _, _, _, _, _, _, _ := GetMockSetup(t)

	tests := []struct {
		name       string
		setup      func(*http.Request)
		assertions func(*httptest.ResponseRecorder, engine.Engine, *engine.User, string, string, string)
	}{
		{
			name:  "Missing Mattermost user ID",
			setup: func(req *http.Request) {},
			assertions: func(rec *httptest.ResponseRecorder, mscal engine.Engine, user *engine.User, eventID, option, postID string) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				assert.Nil(t, mscal)
				assert.Nil(t, user)
				assert.Empty(t, eventID)
				assert.Empty(t, option)
				assert.Empty(t, postID)
			},
		},
		{
			name: "Invalid request body",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				req.Body = io.NopCloser(bytes.NewBufferString("invalid json"))
			},
			assertions: func(rec *httptest.ResponseRecorder, mscal engine.Engine, user *engine.User, eventID, option, postID string) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				assert.Nil(t, mscal)
				assert.Nil(t, user)
				assert.Empty(t, eventID)
				assert.Empty(t, option)
				assert.Empty(t, postID)
			},
		},
		{
			name: "Missing event ID",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{},
					PostId:  MockPostID,
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder, mscal engine.Engine, user *engine.User, eventID, option, postID string) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				assert.Nil(t, mscal)
				assert.Nil(t, user)
				assert.Empty(t, eventID)
				assert.Empty(t, option)
				assert.Empty(t, postID)
			},
		},
		{
			name: "Valid request",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						config.EventIDKey: MockEventID,
						"selected_option": MockOption,
					},
					PostId: MockPostID,
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder, mscal engine.Engine, user *engine.User, eventID, option, postID string) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				assert.NotNil(t, mscal)
				assert.NotNil(t, user)
				assert.Equal(t, MockUserID, user.MattermostUserID)
				assert.Equal(t, MockEventID, eventID)
				assert.Equal(t, MockOption, option)
				assert.Equal(t, MockPostID, postID)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/preprocessAction", nil)
			rec := httptest.NewRecorder()

			tc.setup(req)

			mscal, user, eventID, option, postID := api.preprocessAction(rec, req)

			tc.assertions(rec, mscal, user, eventID, option, postID)
		})
	}
}

func TestPostActionAccept(t *testing.T) {
	api, mockStore, _, mockRemote, mockPluginAPI, _, _, mockClient := GetMockSetup(t)

	tests := []struct {
		name       string
		setup      func(*http.Request)
		assertions func(*httptest.ResponseRecorder)
	}{
		{
			name: "Missing event ID",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
			},
		},
		{
			name: "Accept event successfully",
			setup: func(req *http.Request) {
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(2)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockClient)
				mockPluginAPI.EXPECT().GetMattermostUser(MockUserID).Times(2)
				mockClient.EXPECT().AcceptEvent(MockRemoteUserID, MockEventID).Return(nil)

				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						config.EventIDKey: MockEventID,
					},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/postActionAccept", nil)
			rec := httptest.NewRecorder()

			tc.setup(req)
			api.postActionAccept(rec, req)

			tc.assertions(rec)
		})
	}
}

func TestPostDeclineAccept(t *testing.T) {
	api, mockStore, _, mockRemote, mockPluginAPI, _, _, mockClient := GetMockSetup(t)

	tests := []struct {
		name       string
		setup      func(*http.Request)
		assertions func(*httptest.ResponseRecorder)
	}{
		{
			name: "Missing event ID",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
			},
		},
		{
			name: "Decline event successfully",
			setup: func(req *http.Request) {
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(2)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockClient)
				mockPluginAPI.EXPECT().GetMattermostUser(MockUserID).Times(2)
				mockClient.EXPECT().DeclineEvent(MockRemoteUserID, MockEventID).Return(nil)

				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						config.EventIDKey: MockEventID,
					},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/postActionAccept", nil)
			rec := httptest.NewRecorder()

			tc.setup(req)
			api.postActionDecline(rec, req)

			tc.assertions(rec)
		})
	}
}

func TestPostActionTentative(t *testing.T) {
	api, mockStore, _, mockRemote, mockPluginAPI, _, _, mockClient := GetMockSetup(t)

	tests := []struct {
		name       string
		setup      func(*http.Request)
		assertions func(*httptest.ResponseRecorder)
	}{
		{
			name: "Missing event ID",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
			},
		},
		{
			name: "Tentatively accept event successfully",
			setup: func(req *http.Request) {
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(2)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockClient)
				mockPluginAPI.EXPECT().GetMattermostUser(MockUserID).Times(2)
				mockClient.EXPECT().TentativelyAcceptEvent(MockRemoteUserID, MockEventID).Return(nil)

				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						config.EventIDKey: MockEventID,
					},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/postActionAccept", nil)
			rec := httptest.NewRecorder()

			tc.setup(req)
			api.postActionTentative(rec, req)

			tc.assertions(rec)
		})
	}
}

func TestPostActionRespond(t *testing.T) {
	api, mockStore, _, mockRemote, mockPluginAPI, _, _, mockClient := GetMockSetup(t)

	tests := []struct {
		name       string
		setup      func(*http.Request)
		assertions func(*httptest.ResponseRecorder)
	}{
		{
			name: "Missing event ID",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
			},
		},
		{
			name: "Error responding to event",
			setup: func(req *http.Request) {
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(2)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockClient)
				mockPluginAPI.EXPECT().GetMattermostUser(MockUserID).Return(&model.User{Id: MockUserID}, nil).Times(2)
				mockClient.EXPECT().AcceptEvent(MockRemoteUserID, MockEventID).Return(nil)

				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						config.EventIDKey: MockEventID,
						"selected_option": "decline",
					},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				var response model.PostActionIntegrationResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				assert.Contains(t, response.EphemeralText, "Error: Failed to respond to event")
			},
		},
		{
			name: "Error updating post",
			setup: func(req *http.Request) {
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(2)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockClient)
				mockPluginAPI.EXPECT().GetMattermostUser(MockUserID).Return(&model.User{Id: MockUserID}, nil).Times(2)
				mockPluginAPI.EXPECT().GetPost("").Return(nil, &model.AppError{Message: "error getting post"})
				mockClient.EXPECT().AcceptEvent(MockRemoteUserID, MockEventID).Return(nil)

				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						config.EventIDKey: MockEventID,
						"selected_option": "Yes",
					},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				var response model.PostActionIntegrationResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				assert.Equal(t, response.EphemeralText, "Error: Failed to update the post: error getting post")
			},
		},
		{
			name: "No attachment found",
			setup: func(req *http.Request) {
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(2)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockClient)
				mockPluginAPI.EXPECT().GetMattermostUser(MockUserID).Return(&model.User{Id: MockUserID}, nil).Times(2)
				mockPluginAPI.EXPECT().GetPost("").Return(&model.Post{}, nil)
				mockClient.EXPECT().AcceptEvent(MockRemoteUserID, MockEventID).Return(nil)

				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						config.EventIDKey: MockEventID,
						"selected_option": "Yes",
					},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				var response model.PostActionIntegrationResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				assert.Equal(t, "Error: Failed to update the post: No attachments found", response.EphemeralText)
			},
		},
		{
			name: "Action responded successfully",
			setup: func(req *http.Request) {
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(2)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockClient)
				mockPluginAPI.EXPECT().GetMattermostUser(MockUserID).Return(&model.User{Id: MockUserID}, nil).Times(2)
				mockClient.EXPECT().AcceptEvent(MockRemoteUserID, MockEventID).Return(nil)
				attachment := model.SlackAttachment{
					Title: "Example Title",
					Text:  "This is an example attachment.",
				}
				mockPost := model.Post{
					Id: MockPostID,
					Props: map[string]interface{}{
						"attachments": []*model.SlackAttachment{&attachment},
					},
				}
				mockPluginAPI.EXPECT().GetPost("").Return(&mockPost, nil)

				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						config.EventIDKey: MockEventID,
						"selected_option": "Yes",
					},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/postActionAccept", nil)
			rec := httptest.NewRecorder()

			tc.setup(req)
			api.postActionRespond(rec, req)

			tc.assertions(rec)
		})
	}
}

func TestPostActionConfirmStatusChange(t *testing.T) {
	api, mockStore, _, _, mockPluginAPI, mockLogger, _, _ := GetMockSetup(t)

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
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				var response model.PostActionIntegrationResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response.EphemeralText, "Not authorized.")
			},
		},
		{
			name: "Invalid JSON request body",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				req.Body = io.NopCloser(bytes.NewBufferString("invalid json"))
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				var response model.PostActionIntegrationResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response.EphemeralText, "Invalid request.")
			},
		},
		{
			name: "No recognizable value for property",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				var response model.PostActionIntegrationResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response.EphemeralText, `No recognizable value for property "value".`)
			},
		},
		{
			name: "Error getting user status",
			setup: func(req *http.Request) {
				mockPluginAPI.EXPECT().GetMattermostUserStatus(MockUserID).Return(nil, errors.New("status error")).Times(1)
				mockLogger.EXPECT().Debugf("cannot get user status, err=%s", gomock.Any())

				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						"value":     true,
						"change_to": "away",
					},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				var response model.PostActionIntegrationResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response.EphemeralText, "Cannot get current status.")
			},
		},
		{
			name: "Error loading user",
			setup: func(req *http.Request) {
				mockPluginAPI.EXPECT().GetMattermostUserStatus(MockUserID).Return(&model.Status{Manual: true, Status: "online"}, nil).Times(1)
				mockStore.EXPECT().LoadUser(MockUserID).Return(nil, errors.New("load error")).Times(1)

				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						"value":     true,
						"change_to": "away",
					},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				var response model.PostActionIntegrationResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response.EphemeralText, "Cannot load user")
			},
		},
		{
			name: "Error updating user",
			setup: func(req *http.Request) {
				mockPluginAPI.EXPECT().GetMattermostUserStatus(MockUserID).Return(&model.Status{Manual: true, Status: "online"}, nil).Times(1)
				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{}, nil).Times(1)
				mockStore.EXPECT().StoreUser(gomock.Any()).Return(errors.New("store error")).Times(1)
				mockPluginAPI.EXPECT().UpdateMattermostUserStatus("mockUserID", "away")

				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						"value":     true,
						"change_to": "away",
					},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				var response model.PostActionIntegrationResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response.EphemeralText, "Cannot update user")
			},
		},
		{
			name: "Successful status change",
			setup: func(req *http.Request) {
				mockUserStatus := &model.Status{
					Manual: true,
					Status: "online",
				}

				mockStore.EXPECT().LoadUser(MockUserID).Return(&store.User{Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(1)
				mockStore.EXPECT().StoreUser(gomock.Any()).Return(nil).Times(1)
				mockPluginAPI.EXPECT().GetMattermostUserStatus(MockUserID).Return(mockUserStatus, nil).Times(1)
				mockPluginAPI.EXPECT().UpdateMattermostUserStatus(MockUserID, "away").Times(1)

				req.Header.Set(MMUserIDHeader, MockUserID)
				requestBody := model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						"value":            true,
						"change_to":        "away",
						"pretty_change_to": "Away",
						"hasEvent":         true,
						"subject":          "Meeting with team",
						"weblink":          "http://example.com/meeting",
						"startTime":        time.Now().Format(time.RFC3339),
					},
				}
				bodyBytes, _ := json.Marshal(requestBody)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			},
			assertions: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				var response model.PostActionIntegrationResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				assert.NoError(t, err)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/postActionConfirmStatusChange", nil)
			rec := httptest.NewRecorder()

			tc.setup(req)
			api.postActionConfirmStatusChange(rec, req)

			tc.assertions(rec)
		})
	}
}
