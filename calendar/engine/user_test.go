// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine/mock_plugin_api"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine/mock_welcomer"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote/mock_remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store/mock_store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot/mock_bot"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestExpandUser(t *testing.T) {
	mscalendar, mockStore, _, _, mockPluginAPI, _, _ := GetMockSetup(t)
	mockUser := GetMockUser(nil, nil, MockMMUserID, nil)

	tests := []struct {
		name       string
		setupMock  func()
		assertions func(t *testing.T, err error)
	}{
		{
			name: "error expanding remote user",
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("error filtering the user")).Times(1)
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "error filtering the user")
			},
		},
		{
			name: "error expanding the Mattermost user",
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(&store.User{}, nil).Times(1)
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID).Return(nil, errors.New("some error occurred while getting the Mattermost user"))
			},
			assertions: func(t *testing.T, err error) {
				require.EqualError(t, err, "some error occurred while getting the Mattermost user")
			},
		},
		{
			name: "success expanding the user",
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(&store.User{}, nil).Times(1)
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID).Return(&model.User{}, nil)
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := mscalendar.ExpandUser(mockUser)

			tt.assertions(t, err)
		})
	}
}

func TestExpandRemoteUser(t *testing.T) {
	mscalendar, mockStore, _, _, _, _, _ := GetMockSetup(t)
	mockUser := GetMockUser(nil, nil, MockMMUserID, nil)

	tests := []struct {
		name       string
		setupMock  func()
		assertions func(t *testing.T, err error)
	}{
		{
			name: "error loading the remote user",
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("error filtering the user")).Times(1)
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "error filtering the user")
			},
		},
		{
			name: "success expanding the remote user",
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(&store.User{}, nil).Times(1)
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := mscalendar.ExpandRemoteUser(mockUser)

			tt.assertions(t, err)
		})
	}
}

func TestExpandMattermostUser(t *testing.T) {
	mscalendar, _, _, _, mockPluginAPI, _, _ := GetMockSetup(t)
	mockUser := GetMockUser(nil, nil, MockMMUserID, nil)

	tests := []struct {
		name       string
		setupMock  func()
		assertions func(t *testing.T, err error)
	}{
		{
			name: "error expanding Mattermost user",
			setupMock: func() {
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID).Return(nil, errors.New("some error occurred while getting the Mattermost user")).Times(1)
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.EqualError(t, err, "some error occurred while getting the Mattermost user")
			},
		},
		{
			name: "success expanding the Mattermost user",
			setupMock: func() {
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID).Return(&model.User{}, nil).Times(1)
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := mscalendar.ExpandMattermostUser(mockUser)

			tt.assertions(t, err)
		})
	}
}

func TestGetTimezone(t *testing.T) {
	mscalendar, mockStore, _, _, _, mockClient, _ := GetMockSetup(t)

	tests := []struct {
		name       string
		user       *User
		setupMock  func()
		assertions func(t *testing.T, err error)
	}{
		{
			name: "error loading the remote user",
			user: GetMockUser(nil, nil, MockMMUserID, nil),
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("error filtering the user")).Times(1)
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "error filtering the user")
			},
		},
		{
			name: "error getting the mailbox setting",
			user: GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, GetMockStoreSettings()),
			setupMock: func() {
				mockClient.EXPECT().GetMailboxSettings(MockRemoteUserID).Return(nil, errors.New("error occurred while getting the mailbox settings")).Times(1)
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.EqualError(t, err, "error occurred while getting the mailbox settings")
			},
		},
		{
			name: "success getting mailbox setting",
			user: GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, GetMockStoreSettings()),
			setupMock: func() {
				mockClient.EXPECT().GetMailboxSettings(MockRemoteUserID).Return(&remote.MailboxSettings{TimeZone: MockTimeZone}, nil).Times(1)
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			_, err := mscalendar.GetTimezone(tt.user)

			tt.assertions(t, err)
		})
	}
}

func TestUser_String(t *testing.T) {
	tests := []struct {
		name       string
		user       *User
		assertions func(t *testing.T, actualString string)
	}{
		{
			name: "User with the Mattermost user object",
			user: GetMockUserWithDefaultDailySummaryUserSettings(),
			assertions: func(t *testing.T, actualString string) {
				require.Equal(t, fmt.Sprintf("@%s", MockMMUsername), actualString)
			},
		},
		{
			name: "User without the Mattermost user object",
			user: &User{
				MattermostUserID: MockMMUserID,
			},
			assertions: func(t *testing.T, actualString string) {
				require.Equal(t, MockMMUserID, actualString)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualString := tt.user.String()

			tt.assertions(t, actualString)
		})
	}
}

func TestUser_Markdown(t *testing.T) {
	tests := []struct {
		name       string
		user       *User
		assertions func(t *testing.T, actualOutput string)
	}{
		{
			name: "User with the Mattermost user object",
			user: GetMockUserWithDefaultDailySummaryUserSettings(),
			assertions: func(t *testing.T, actualOutput string) {
				require.Equal(t, fmt.Sprintf("@%s", MockMMUsername), actualOutput)
			},
		},
		{
			name: "User without the Mattermost user object",
			user: &User{
				MattermostUserID: MockMMUserID,
			},
			assertions: func(t *testing.T, actualOutput string) {
				require.Equal(t, "UserID: `testMMUserID`", actualOutput)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualOutput := tt.user.Markdown()

			tt.assertions(t, actualOutput)
		})
	}
}

func TestDisconnectUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockPoster := mock_bot.NewMockPoster(ctrl)
	mockRemote := mock_remote.NewMockRemote(ctrl)
	mockPluginAPI := mock_plugin_api.NewMockPluginAPI(ctrl)
	mockClient := mock_remote.NewMockClient(ctrl)
	mockLogger := mock_bot.NewMockLogger(ctrl)
	mockLoggerWith := mock_bot.NewMockLogger(ctrl)
	mockWelcomer := mock_welcomer.NewMockWelcomer(ctrl)
	env := Env{
		Dependencies: &Dependencies{
			Store:     mockStore,
			Poster:    mockPoster,
			Remote:    mockRemote,
			PluginAPI: mockPluginAPI,
			Logger:    mockLogger,
			Welcomer:  mockWelcomer,
		},
	}
	mscalendar := &mscalendar{
		Env:    env,
		client: mockClient,
	}
	mscalendar.Config = &config.Config{
		Provider: config.ProviderConfig{
			DisplayName:    "testDisplayName",
			CommandTrigger: "testCommandTrigger",
		},
	}

	tests := []struct {
		name       string
		setupMock  func()
		assertions func(err error)
	}{
		{
			name: "error filtering the user",
			setupMock: func() {
				mscalendar.client = nil
				mscalendar.actingUser = &User{MattermostUserID: MockRemoteUserID}
				mockWelcomer.EXPECT().AfterDisconnect(MockMMUserID).Return(nil).Times(1)
				mockStore.EXPECT().LoadUser(MockRemoteUserID).Return(nil, errors.New("error filtering the user")).Times(1)
			},
			assertions: func(err error) {
				require.ErrorContains(t, err, "error filtering the user")
			},
		},
		{
			name: "error loading the user",
			setupMock: func() {
				mscalendar.client = mockClient
				mscalendar.actingUser = &User{MattermostUserID: MockRemoteUserID}
				mockWelcomer.EXPECT().AfterDisconnect(MockMMUserID).Return(nil).Times(1)
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("error loading the user")).Times(1)
			},
			assertions: func(err error) {
				require.EqualError(t, err, "error loading the user")
			},
		},
		{
			name: "error deleting the linked channels from events",
			setupMock: func() {
				mscalendar.client = mockClient
				mscalendar.actingUser = &User{MattermostUserID: MockRemoteUserID}
				mockWelcomer.EXPECT().AfterDisconnect(MockMMUserID).Return(nil).Times(1)
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(&store.User{ChannelEvents: store.ChannelEventLink{MockEventID: mockChannelID}, MattermostDisplayName: MockMMUserDisplayName}, nil).Times(1)
				mockStore.EXPECT().DeleteLinkedChannelFromEvent(MockEventID, mockChannelID).Return(errors.New("some error occurred deleting linked channel")).Times(1)
				mockStore.EXPECT().StoreUser(gomock.Any()).Return(errors.New("some error occurred storing user")).Times(1)
				mockLogger.EXPECT().With(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Errorf("error storing user after failing deleting linked channels from store").Times(1)
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.EqualError(t, err, "error deleting linked channels from events")
			},
		},
		{
			name: "error loading the subscription",
			setupMock: func() {
				mscalendar.client = mockClient
				mscalendar.actingUser = &User{MattermostUserID: MockRemoteUserID}
				mockWelcomer.EXPECT().AfterDisconnect(MockMMUserID).Return(nil).Times(1)
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(&store.User{Settings: store.Settings{EventSubscriptionID: MockEventSubscriptionID}}, nil).Times(1)
				mockStore.EXPECT().LoadSubscription(MockEventSubscriptionID).Return(nil, errors.New("internal error")).Times(1)
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.EqualError(t, err, "error loading subscription: internal error")
			},
		},
		{
			name: "failed to delete event subscription",
			setupMock: func() {
				mscalendar.client = mockClient
				mscalendar.actingUser = &User{MattermostUserID: MockRemoteUserID}
				mockWelcomer.EXPECT().AfterDisconnect(MockMMUserID).Return(nil).Times(1)
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(&store.User{Settings: store.Settings{EventSubscriptionID: MockEventSubscriptionID}}, nil).Times(1)
				mockStore.EXPECT().LoadSubscription(MockEventSubscriptionID).Return(nil, nil).Times(1)
				mockStore.EXPECT().DeleteUserSubscription(gomock.Any(), MockEventSubscriptionID).Return(errors.New("internal server error")).Times(1)
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.EqualError(t, err, "failed to delete subscription testEventSubscriptionID: internal server error")
			},
		},
		{
			name: "error deleting user",
			setupMock: func() {
				mscalendar.client = mockClient
				mscalendar.actingUser = &User{MattermostUserID: MockRemoteUserID}
				mockWelcomer.EXPECT().AfterDisconnect(MockMMUserID).Return(nil).Times(1)
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(&store.User{Settings: store.Settings{EventSubscriptionID: MockEventSubscriptionID}}, nil).Times(1)
				mockStore.EXPECT().LoadSubscription(MockEventSubscriptionID).Return(&store.Subscription{Remote: &remote.Subscription{}}, nil).Times(1)
				mockStore.EXPECT().DeleteUserSubscription(gomock.Any(), MockEventSubscriptionID).Return(nil).Times(1)
				mockClient.EXPECT().DeleteSubscription(gomock.Any()).Return(nil).Times(1)
				mockStore.EXPECT().DeleteUser(MockMMUserID).Return(errors.New("error deleting user")).Times(1)
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.EqualError(t, err, "error deleting user")
			},
		},
		{
			name: "error deleting user from index",
			setupMock: func() {
				mscalendar.client = mockClient
				mscalendar.actingUser = &User{MattermostUserID: MockRemoteUserID}
				mockWelcomer.EXPECT().AfterDisconnect(MockMMUserID).Return(nil).Times(1)
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(&store.User{Settings: store.Settings{EventSubscriptionID: MockEventSubscriptionID}}, nil).Times(1)
				mockStore.EXPECT().LoadSubscription(MockEventSubscriptionID).Return(&store.Subscription{Remote: &remote.Subscription{}}, nil).Times(1)
				mockStore.EXPECT().DeleteUserSubscription(gomock.Any(), MockEventSubscriptionID).Return(nil).Times(1)
				mockClient.EXPECT().DeleteSubscription(gomock.Any()).Return(nil).Times(1)
				mockStore.EXPECT().DeleteUser(MockMMUserID).Return(nil).Times(1)
				mockStore.EXPECT().DeleteUserFromIndex(MockMMUserID).Return(errors.New("error deleting user from index")).Times(1)
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.EqualError(t, err, "error deleting user from index")
			},
		},
		{
			name: "user disconnected successfully",
			setupMock: func() {
				mscalendar.client = mockClient
				mscalendar.actingUser = &User{MattermostUserID: MockRemoteUserID}
				mockWelcomer.EXPECT().AfterDisconnect(MockMMUserID).Return(nil).Times(1)
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(&store.User{Settings: store.Settings{EventSubscriptionID: MockEventSubscriptionID}}, nil).Times(1)
				mockStore.EXPECT().LoadSubscription(MockEventSubscriptionID).Return(&store.Subscription{Remote: &remote.Subscription{}}, nil).Times(1)
				mockStore.EXPECT().DeleteUserSubscription(gomock.Any(), MockEventSubscriptionID).Return(nil).Times(1)
				mockClient.EXPECT().DeleteSubscription(gomock.Any()).Return(nil).Times(1)
				mockStore.EXPECT().DeleteUser(MockMMUserID).Return(nil).Times(1)
				mockStore.EXPECT().DeleteUserFromIndex(MockMMUserID).Return(nil).Times(1)
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := mscalendar.DisconnectUser(MockMMUserID)

			tt.assertions(err)
		})
	}
}

func TestGetRemoteUser(t *testing.T) {
	mscalendar, mockStore, _, _, _, _, _ := GetMockSetup(t)

	tests := []struct {
		name       string
		setupMock  func()
		assertions func(remoteUser *remote.User, err error)
	}{
		{
			name: "Error loading user",
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("failed to load user")).Times(1)
			},
			assertions: func(remoteUser *remote.User, err error) {
				require.Error(t, err)
				require.EqualError(t, err, "failed to load user")
				require.Nil(t, remoteUser)
			},
		},
		{
			name: "Successfully get remote user",
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(&store.User{Remote: &remote.User{ID: MockRemoteUserID}}, nil).Times(1)
			},
			assertions: func(remoteUser *remote.User, err error) {
				require.NoError(t, err)
				require.Equal(t, &remote.User{ID: MockRemoteUserID}, remoteUser)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			remoteUser, err := mscalendar.GetRemoteUser(MockMMUserID)

			tt.assertions(remoteUser, err)
		})
	}
}

func TestIsAuthorizedAdmin(t *testing.T) {
	mscalendar, _, _, _, mockPluginAPI, _, _ := GetMockSetup(t)

	tests := []struct {
		name             string
		mattermostUserID string
		setupMock        func()
		assertions       func(result bool, err error)
	}{
		{
			name:             "User is in AdminUserIDs",
			mattermostUserID: "mockAdminID1",
			setupMock: func() {
				mscalendar.AdminUserIDs = "mockAdminID1,mockAdminID2"
			},
			assertions: func(result bool, err error) {
				require.NoError(t, err)
				require.Equal(t, true, result)
			},
		},
		{
			name:             "error checking system admin",
			mattermostUserID: MockMMUserID,
			setupMock: func() {
				mscalendar.AdminUserIDs = "mockAdminID1,mockAdminID2"
				mockPluginAPI.EXPECT().IsSysAdmin(MockMMUserID).Return(false, errors.New("error occurred checking system admin")).Times(1)
			},
			assertions: func(result bool, err error) {
				require.Error(t, err)
				require.EqualError(t, err, "error occurred checking system admin")
			},
		},
		{
			name:             "User is not in AdminUserIDs and is not a system admin",
			mattermostUserID: MockMMUserID,
			setupMock: func() {
				mscalendar.AdminUserIDs = "mockAdminID1,mockAdminID2"
				mockPluginAPI.EXPECT().IsSysAdmin(MockMMUserID).Return(false, nil).Times(1)
			},
			assertions: func(result bool, err error) {
				require.NoError(t, err)
				require.Equal(t, false, result)
			},
		},
		{
			name:             "User is not in AdminUserIDs but is a system admin",
			mattermostUserID: MockMMUserID,
			setupMock: func() {
				mscalendar.AdminUserIDs = "mockAdminID1,mockAdminID2"
				mockPluginAPI.EXPECT().IsSysAdmin(MockMMUserID).Return(true, nil).Times(1)
			},
			assertions: func(result bool, err error) {
				require.NoError(t, err)
				require.Equal(t, true, result)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := mscalendar.IsAuthorizedAdmin(tt.mattermostUserID)

			tt.assertions(result, err)
		})
	}
}

func TestGetUserSettings(t *testing.T) {
	mscalendar, mockStore, _, _, mockPluginAPI, _, _ := GetMockSetup(t)
	mockUser := GetMockUser(nil, nil, MockMMUserID, nil)

	tests := []struct {
		name       string
		setupMock  func()
		assertions func(result *store.Settings, err error)
	}{
		{
			name: "error filtering the user",
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("error filtering user")).Times(1)
			},
			assertions: func(result *store.Settings, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "error filtering user")
			},
		},
		{
			name: "Successfully get user settings",
			setupMock: func() {
				mockUser.User = &store.User{Settings: store.Settings{GetConfirmation: false}, Remote: &remote.User{ID: MockRemoteID}}
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID).Return(&model.User{}, nil).Times(1)
			},
			assertions: func(result *store.Settings, err error) {
				require.NoError(t, err)
				require.Equal(t, &store.Settings{GetConfirmation: false}, result)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := mscalendar.GetUserSettings(mockUser)

			tt.assertions(result, err)
		})
	}
}
