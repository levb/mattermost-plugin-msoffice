// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine/mock_plugin_api"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote/mock_remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store/mock_store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot/mock_bot"

	"github.com/mattermost/mattermost/server/public/model"
)

const (
	MockRemoteUserID      = "testRemoteUserID"
	MockMMModelUserID     = "testMMModelUserID"
	MockMMUserID          = "testMMUserID"
	MockMMUsername        = "testMMUsername"
	MockMMUserDisplayName = "testMMUserDisplayName"
	MockRemoteID          = "testRemoteID"

	MockCalendarName = "Test Calendar"
	MockCalendarID   = "testCalendarID"

	MockEventName           = "Test Event"
	MockEventID             = "testEventID"
	MockEventSubscriptionID = "testEventSubscriptionID"

	MockActingUserID       = "testActingUserID"
	MockActingUserRemoteID = "testActingUserRemoteID"

	MockTimeZone  = "testTimeZone"
	mockChannelID = "testChannelID"
)

// revive:disable:unexported-return
func GetMockSetup(t *testing.T) (*mscalendar, *mock_store.MockStore, *mock_bot.MockPoster, *mock_remote.MockRemote, *mock_plugin_api.MockPluginAPI, *mock_remote.MockClient, *mock_bot.MockLogger) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock_store.NewMockStore(ctrl)
	mockPoster := mock_bot.NewMockPoster(ctrl)
	mockRemote := mock_remote.NewMockRemote(ctrl)
	mockPluginAPI := mock_plugin_api.NewMockPluginAPI(ctrl)
	mockClient := mock_remote.NewMockClient(ctrl)
	mockLogger := mock_bot.NewMockLogger(ctrl)

	env := Env{
		Dependencies: &Dependencies{
			Store:     mockStore,
			Poster:    mockPoster,
			Remote:    mockRemote,
			PluginAPI: mockPluginAPI,
			Logger:    mockLogger,
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
		PluginVersion: "1.0.0",
	}

	return mscalendar, mockStore, mockPoster, mockRemote, mockPluginAPI, mockClient, mockLogger
}

func GetMockUser(remoteUserID, mmModelUserID *string, mmUserID string, storeUserSetting *store.Settings) *User {
	user := (*store.User)(nil)
	switch {
	case remoteUserID != nil && storeUserSetting != nil:
		user = &store.User{
			Remote:   &remote.User{ID: *remoteUserID},
			Settings: *storeUserSetting,
		}
	case remoteUserID != nil:
		user = &store.User{
			Remote: &remote.User{ID: *remoteUserID},
		}
	case storeUserSetting != nil:
		user = &store.User{
			Settings: *storeUserSetting,
		}
	}

	mmUser := (*model.User)(nil)
	if mmModelUserID != nil {
		mmUser = &model.User{Id: *mmModelUserID}
	}

	return &User{
		User:             user,
		MattermostUser:   mmUser,
		MattermostUserID: mmUserID,
	}
}

func GetMockUserWithDefaultDailySummaryUserSettings() *User {
	return &User{
		MattermostUserID: MockMMUserID,
		MattermostUser: &model.User{
			Id:       MockMMModelUserID,
			Username: MockMMUsername,
		},
		User: &store.User{
			Remote: &remote.User{
				ID: MockRemoteUserID,
			},
			Settings: store.Settings{
				DailySummary: store.DefaultDailySummaryUserSettings(),
			},
		},
	}
}

func GetMockCalendar(name string) *remote.Calendar {
	return &remote.Calendar{
		Name: name,
	}
}

func GetMockEvent(subject string, location *remote.Location, start, end *remote.DateTime, attendees []*remote.Attendee) *remote.Event {
	return &remote.Event{
		Subject:   subject,
		Location:  location,
		Start:     start,
		End:       end,
		Attendees: attendees,
	}
}

func GetMockStoreSettings() *store.Settings {
	return &store.Settings{}
}

func GetMockSubscription() *store.Subscription {
	return &store.Subscription{
		Remote:              &remote.Subscription{},
		MattermostCreatorID: MockActingUserID,
		PluginVersion:       "1.0.0",
	}
}
