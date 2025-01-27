// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestAcceptEvent(t *testing.T) {
	mscalendar, mockStore, _, _, mockPluginAPI, mockClient, _ := GetMockSetup(t)

	tests := []struct {
		name      string
		user      *User
		setupMock func()
		assertion func(err error)
	}{
		{
			name: "error filtering with user",
			user: GetMockUser(nil, nil, MockMMUserID, nil),
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("error filtering the user")).Times(1)
			},
			assertion: func(err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "error filtering the user")
			},
		},
		{
			name: "error accepting the event",
			user: GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, GetMockStoreSettings()),
			setupMock: func() {
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID)
				mockClient.EXPECT().AcceptEvent(MockRemoteUserID, MockEventID).Return(errors.New("unable to accept the event")).Times(1)
			},
			assertion: func(err error) {
				require.Error(t, err)
				require.EqualError(t, err, "unable to accept the event")
			},
		},
		{
			name: "successful event acceptance",
			user: GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, GetMockStoreSettings()),
			setupMock: func() {
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID)
				mockClient.EXPECT().AcceptEvent(MockRemoteUserID, MockEventID).Return(nil).Times(1)
			},
			assertion: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := mscalendar.AcceptEvent(tt.user, MockEventID)

			tt.assertion(err)
		})
	}
}

func TestDeclineEvent(t *testing.T) {
	mscalendar, mockStore, _, _, mockPluginAPI, mockClient, _ := GetMockSetup(t)

	tests := []struct {
		name      string
		user      *User
		setupMock func()
		assertion func(err error)
	}{
		{
			name: "error filtering with user",
			user: GetMockUser(nil, nil, MockMMUserID, nil),
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("error filtering user")).Times(1)
			},
			assertion: func(err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "error filtering user")
			},
		},
		{
			name: "error declining event",
			user: GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, GetMockStoreSettings()),
			setupMock: func() {
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID)
				mockClient.EXPECT().DeclineEvent(MockRemoteUserID, MockEventID).Return(errors.New("unable to decline event")).Times(1)
			},
			assertion: func(err error) {
				require.Error(t, err)
				require.EqualError(t, err, "unable to decline event")
			},
		},
		{
			name: "successful event decline",
			user: GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, GetMockStoreSettings()),
			setupMock: func() {
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID)
				mockClient.EXPECT().DeclineEvent(MockRemoteUserID, MockEventID).Return(nil).Times(1)
			},
			assertion: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := mscalendar.DeclineEvent(tt.user, MockEventID)

			tt.assertion(err)
		})
	}
}

func TestTentativelyAcceptEvent(t *testing.T) {
	mscalendar, mockStore, _, _, mockPluginAPI, mockClient, _ := GetMockSetup(t)

	tests := []struct {
		name      string
		user      *User
		setupMock func()
		assertion func(err error)
	}{
		{
			name: "error filtering with the user",
			user: GetMockUser(nil, nil, MockMMUserID, nil),
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("error filtering the user")).Times(1)
			},
			assertion: func(err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "error filtering the user")
			},
		},
		{
			name: "error tentatively accepting the event",
			user: GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, GetMockStoreSettings()),
			setupMock: func() {
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID)
				mockClient.EXPECT().TentativelyAcceptEvent(MockRemoteUserID, MockEventID).Return(errors.New("unable to tentatively accept the event")).Times(1)
			},
			assertion: func(err error) {
				require.Error(t, err)
				require.EqualError(t, err, "unable to tentatively accept the event")
			},
		},
		{
			name: "successful tentative event acceptance",
			user: GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, GetMockStoreSettings()),
			setupMock: func() {
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID)
				mockClient.EXPECT().TentativelyAcceptEvent(MockRemoteUserID, MockEventID).Return(nil).Times(1)
			},
			assertion: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := mscalendar.TentativelyAcceptEvent(tt.user, MockEventID)

			tt.assertion(err)
		})
	}
}

func TestRespondToEvent(t *testing.T) {
	mscalendar, mockStore, _, _, mockPluginAPI, mockClient, _ := GetMockSetup(t)

	tests := []struct {
		name      string
		response  string
		user      *User
		setupMock func()
		assertion func(err error)
	}{
		{
			name:      "invalid response error",
			response:  OptionNotResponded,
			user:      GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, GetMockStoreSettings()),
			setupMock: func() {},
			assertion: func(err error) {
				require.Error(t, err)
				require.EqualError(t, err, "not responded is not a valid response")
			},
		},
		{
			name:     "invalid response string",
			response: "InvalidResponse",
			user:     GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, GetMockStoreSettings()),
			setupMock: func() {
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID).Return(&model.User{Id: MockMMUserID}, nil)
			},
			assertion: func(err error) {
				require.EqualError(t, err, "InvalidResponse is not a valid response")
			},
		},
		{
			name:     "error filtering the user",
			response: OptionYes,
			user:     GetMockUser(nil, nil, MockMMUserID, nil),
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("error filtering the user")).Times(1)
			},
			assertion: func(err error) {
				require.ErrorContains(t, err, "error filtering the user")
			},
		},
		{
			name:     "success accepting the event",
			response: OptionYes,
			user:     GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, GetMockStoreSettings()),
			setupMock: func() {
				mockClient.EXPECT().AcceptEvent(MockRemoteUserID, MockEventID).Return(nil).Times(1)
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID).Return(&model.User{Id: MockMMUserID}, nil)
			},
			assertion: func(err error) {
				require.NoError(t, err)
			},
		},
		{
			name:     "error accepting the event",
			response: OptionYes,
			user:     GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, GetMockStoreSettings()),
			setupMock: func() {
				mockClient.EXPECT().AcceptEvent(MockRemoteUserID, MockEventID).Return(errors.New("unable to accept the event")).Times(1)
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID).Return(&model.User{Id: MockMMUserID}, nil)
			},
			assertion: func(err error) {
				require.Error(t, err)
				require.EqualError(t, err, "unable to accept the event")
			},
		},
		{
			name:     "success declining the event",
			response: OptionNo,
			user:     GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, GetMockStoreSettings()),
			setupMock: func() {
				mockClient.EXPECT().DeclineEvent(MockRemoteUserID, MockEventID).Return(nil).Times(1)
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID).Return(&model.User{Id: MockMMUserID}, nil)
			},
			assertion: func(err error) {
				require.NoError(t, err)
			},
		},
		{
			name:     "error declining the event",
			response: OptionNo,
			user:     GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, GetMockStoreSettings()),
			setupMock: func() {
				mockClient.EXPECT().DeclineEvent(MockRemoteUserID, MockEventID).Return(errors.New("unable to decline the event")).Times(1)
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID).Return(&model.User{Id: MockMMUserID}, nil)
			},
			assertion: func(err error) {
				require.Error(t, err)
				require.EqualError(t, err, "unable to decline the event")
			},
		},
		{
			name:     "success tentatively accepting the event",
			response: OptionMaybe,
			user:     GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, GetMockStoreSettings()),
			setupMock: func() {
				mockClient.EXPECT().TentativelyAcceptEvent(MockRemoteUserID, MockEventID).Return(nil).Times(1)
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID).Return(&model.User{Id: MockMMUserID}, nil)
			},
			assertion: func(err error) {
				require.NoError(t, err)
			},
		},
		{
			name:     "error tentatively accepting the event",
			response: OptionMaybe,
			user:     GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, GetMockStoreSettings()),
			setupMock: func() {
				mockClient.EXPECT().TentativelyAcceptEvent(MockRemoteUserID, MockEventID).Return(errors.New("unable to tentatively accept the event")).Times(1)
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID).Return(&model.User{Id: MockMMUserID}, nil)
			},
			assertion: func(err error) {
				require.Error(t, err)
				require.EqualError(t, err, "unable to tentatively accept the event")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := mscalendar.RespondToEvent(tt.user, MockEventID, tt.response)

			tt.assertion(err)
		})
	}
}
