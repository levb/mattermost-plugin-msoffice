// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine/mock_plugin_api"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote/mock_remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store/mock_store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot/mock_bot"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/test"
)

func TestSyncStatusAll(t *testing.T) {
	moment := time.Now().UTC()
	eventHash := "event_id " + moment.Format(time.RFC3339)
	busyEvent := &remote.Event{ICalUID: "event_id", Start: remote.NewDateTime(moment, "UTC"), ShowAs: "busy", Attendees: []*remote.Attendee{{
		EmailAddress: &remote.EmailAddress{
			Address: "mock-attendee@gmail.com",
		},
	}}}

	for name, tc := range map[string]struct {
		apiError            *remote.APIError
		currentStatus       string
		newStatus           string
		remoteEvents        []*remote.Event
		activeEvents        []string
		eventsToStore       []string
		currentStatusManual bool
		shouldLogError      bool
		getConfirmation     bool
	}{
		"Most common case, no events local or remote. No status change.": {
			remoteEvents:        []*remote.Event{},
			activeEvents:        []string{},
			currentStatus:       "online",
			currentStatusManual: true,
			newStatus:           "",
			eventsToStore:       nil,
			shouldLogError:      false,
			getConfirmation:     false,
		},
		"New remote event. Change status to DND.": {
			remoteEvents:        []*remote.Event{busyEvent},
			activeEvents:        []string{},
			currentStatus:       "online",
			currentStatusManual: true,
			newStatus:           "dnd",
			eventsToStore:       []string{eventHash},
			shouldLogError:      false,
			getConfirmation:     false,
		},
		"Locally stored event is finished. Change status to online.": {
			remoteEvents:        []*remote.Event{},
			activeEvents:        []string{eventHash},
			currentStatus:       "dnd",
			currentStatusManual: true,
			newStatus:           "online",
			eventsToStore:       []string{},
			shouldLogError:      false,
			getConfirmation:     false,
		},
		"Locally stored event is still happening. No status change.": {
			remoteEvents:        []*remote.Event{busyEvent},
			activeEvents:        []string{eventHash},
			currentStatus:       "dnd",
			currentStatusManual: true,
			newStatus:           "",
			eventsToStore:       nil,
			shouldLogError:      false,
			getConfirmation:     false,
		},
		"User has manually set themselves to online during event. Locally stored event is still happening, but we will ignore it. No status change.": {
			remoteEvents:        []*remote.Event{busyEvent},
			activeEvents:        []string{eventHash},
			currentStatus:       "online",
			currentStatusManual: true,
			newStatus:           "",
			eventsToStore:       nil,
			shouldLogError:      false,
			getConfirmation:     false,
		},
		"Ignore non-busy event": {
			remoteEvents:        []*remote.Event{{ID: "event_id_2", Start: remote.NewDateTime(moment, "UTC"), ShowAs: "free"}},
			activeEvents:        []string{},
			currentStatus:       "online",
			currentStatusManual: true,
			newStatus:           "",
			eventsToStore:       nil,
			shouldLogError:      false,
			getConfirmation:     false,
		},
		"Remote API error. Error should be logged": {
			remoteEvents:        nil,
			activeEvents:        []string{eventHash},
			currentStatus:       "online",
			currentStatusManual: true,
			newStatus:           "",
			eventsToStore:       nil,
			apiError:            &remote.APIError{Code: "403", Message: "Forbidden"},
			shouldLogError:      true,
			getConfirmation:     false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			env, client := makeStatusSyncTestEnv(ctrl)
			deps := env.Dependencies

			c, r, papi, s, logger := client.(*mock_remote.MockClient), env.Remote.(*mock_remote.MockRemote), deps.PluginAPI.(*mock_plugin_api.MockPluginAPI), deps.Store.(*mock_store.MockStore), deps.Logger.(*mock_bot.MockLogger)
			s.EXPECT().LoadUserIndex().Return(store.UserIndex{
				&store.UserShort{
					MattermostUserID: "user_mm_id",
					RemoteID:         "user_remote_id",
					Email:            "user_email@example.com",
				},
			}, nil).Times(1)
			r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)

			mockUser := &store.User{
				MattermostUserID: "user_mm_id",
				Remote: &remote.User{
					ID:   "user_remote_id",
					Mail: "user_email@example.com",
				},
				Settings: store.Settings{
					UpdateStatusFromOptions: store.DNDStatusOption,
					GetConfirmation:         tc.getConfirmation,
					SetCustomStatus:         false,
				},
				ActiveEvents: tc.activeEvents,
			}
			s.EXPECT().LoadUser("user_mm_id").Return(mockUser, nil).Times(1)

			c.EXPECT().DoBatchViewCalendarRequests(gomock.Any()).Return([]*remote.ViewCalendarResponse{
				{Events: tc.remoteEvents, RemoteUserID: "user_remote_id", Error: tc.apiError},
			}, nil)

			papi.EXPECT().GetMattermostUserStatusesByIds([]string{"user_mm_id"}).Return([]*model.Status{{Status: tc.currentStatus, Manual: tc.currentStatusManual, UserId: "user_mm_id"}}, nil)

			if tc.newStatus == "" {
				papi.EXPECT().UpdateMattermostUserStatus("user_mm_id", gomock.Any()).Times(0)
			} else {
				if tc.currentStatusManual && !tc.getConfirmation ||
					tc.currentStatusManual && tc.currentStatus == "dnd" {
					if tc.newStatus == "dnd" {
						mockUser.LastStatus = tc.currentStatus
					}
					s.EXPECT().StoreUser(mockUser).Return(nil).Times(1)
				}
				papi.EXPECT().UpdateMattermostUserStatus("user_mm_id", tc.newStatus).Return(nil, nil)
			}

			if tc.eventsToStore == nil {
				s.EXPECT().StoreUserActiveEvents("user_mm_id", gomock.Any()).Return(nil).Times(0)
			} else {
				s.EXPECT().StoreUserActiveEvents("user_mm_id", tc.eventsToStore).Return(nil).Times(1)
			}

			if tc.shouldLogError {
				logger.EXPECT().Warnf("Error getting availability for %s. err=%s", "user_mm_id", tc.apiError.Message).Times(1)
			} else {
				logger.EXPECT().Warnf(gomock.Any()).Times(0)
			}

			m := New(env, "")
			res, _, err := m.SyncAll()
			require.Nil(t, err)
			require.NotEmpty(t, res)
		})
	}
}

func TestSyncStatusUserConfig(t *testing.T) {
	for name, tc := range map[string]struct {
		runAssertions func(deps *Dependencies, client remote.Client)
		settings      store.Settings
	}{
		"UpdateStatusFromOptions default": {
			settings: store.Settings{
				UpdateStatusFromOptions: store.NotSetStatusOption,
			},
			runAssertions: func(deps *Dependencies, client remote.Client) {
				c, r := client.(*mock_remote.MockClient), deps.Remote.(*mock_remote.MockRemote)
				c.EXPECT().DoBatchViewCalendarRequests(gomock.Any()).Times(0)
				r.EXPECT().MakeSuperuserClient(gomock.Any())
			},
		},
		"UpdateStatusFromOptions away and GetConfirmation enabled": {
			settings: store.Settings{
				UpdateStatusFromOptions: store.AwayStatusOption,
				GetConfirmation:         true,
			},
			runAssertions: func(deps *Dependencies, client remote.Client) {
				c, r, papi, poster, s := client.(*mock_remote.MockClient), deps.Remote.(*mock_remote.MockRemote), deps.PluginAPI.(*mock_plugin_api.MockPluginAPI), deps.Poster.(*mock_bot.MockPoster), deps.Store.(*mock_store.MockStore)
				r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)
				moment := time.Now().UTC()
				busyEvent := &remote.Event{ICalUID: "event_id", Start: remote.NewDateTime(moment, "UTC"), ShowAs: "busy", Attendees: []*remote.Attendee{{EmailAddress: &remote.EmailAddress{Address: "mock-attendee@gmail.com"}}}}

				c.EXPECT().DoBatchViewCalendarRequests(gomock.Any()).Times(1).Return([]*remote.ViewCalendarResponse{
					{Events: []*remote.Event{busyEvent}, RemoteUserID: "user_remote_id"},
				}, nil)
				papi.EXPECT().GetMattermostUserStatusesByIds([]string{"user_mm_id"}).Return([]*model.Status{{Status: "online", Manual: true, UserId: "user_mm_id"}}, nil)

				s.EXPECT().StoreUser(gomock.Any()).Return(nil).Times(1)
				s.EXPECT().StoreUserActiveEvents("user_mm_id", []string{"event_id " + moment.Format(time.RFC3339)})
				poster.EXPECT().DMWithAttachments("user_mm_id", gomock.Any()).Times(1)
				papi.EXPECT().UpdateMattermostUserStatus("user_mm_id", gomock.Any()).Times(0)
			},
		},
		"UpdateStatusFromOptions do not disturb, GetConfirmation enabled and no attendee present": {
			settings: store.Settings{
				UpdateStatusFromOptions: store.DNDStatusOption,
				GetConfirmation:         true,
			},
			runAssertions: func(deps *Dependencies, client remote.Client) {
				c, papi, _, _, r := client.(*mock_remote.MockClient), deps.PluginAPI.(*mock_plugin_api.MockPluginAPI), deps.Poster.(*mock_bot.MockPoster), deps.Store.(*mock_store.MockStore), deps.Remote.(*mock_remote.MockRemote)
				moment := time.Now().UTC()
				busyEvent := &remote.Event{ICalUID: "event_id", Start: remote.NewDateTime(moment, "UTC"), ShowAs: "busy"}

				c.EXPECT().DoBatchViewCalendarRequests(gomock.Any()).Times(1).Return([]*remote.ViewCalendarResponse{
					{Events: []*remote.Event{busyEvent}, RemoteUserID: "user_remote_id"},
				}, nil)
				papi.EXPECT().GetMattermostUserStatusesByIds([]string{"user_mm_id"}).Return([]*model.Status{{Status: "online", Manual: true, UserId: "user_mm_id"}}, nil)

				papi.EXPECT().UpdateMattermostUserStatus("user_mm_id", gomock.Any()).Times(0)

				r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)
			},
		},
		"UpdateStatusFromOptions do not disturb, GetConfirmation enabled and overlapping events present": {
			settings: store.Settings{
				UpdateStatusFromOptions: store.DNDStatusOption,
				GetConfirmation:         true,
			},
			runAssertions: func(deps *Dependencies, client remote.Client) {
				c, papi, _, _, r := client.(*mock_remote.MockClient), deps.PluginAPI.(*mock_plugin_api.MockPluginAPI), deps.Poster.(*mock_bot.MockPoster), deps.Store.(*mock_store.MockStore), deps.Remote.(*mock_remote.MockRemote)
				moment := time.Now().UTC()
				firstBusyEvent := &remote.Event{ICalUID: "event_id-1", Start: remote.NewDateTime(moment, "UTC"), End: remote.NewDateTime(moment.Add(10*time.Minute), "UTC"), ShowAs: "busy"}
				secondBusyEvent := &remote.Event{ICalUID: "event_id-2", Start: remote.NewDateTime(moment.Add(5*time.Minute), "UTC"), End: remote.NewDateTime(moment.Add(15*time.Minute), "UTC"), ShowAs: "busy"}

				c.EXPECT().DoBatchViewCalendarRequests(gomock.Any()).Times(1).Return([]*remote.ViewCalendarResponse{
					{Events: []*remote.Event{firstBusyEvent, secondBusyEvent}, RemoteUserID: "user_remote_id"},
				}, nil)
				papi.EXPECT().GetMattermostUserStatusesByIds([]string{"user_mm_id"}).Return([]*model.Status{{Status: "online", Manual: true, UserId: "user_mm_id"}}, nil)

				papi.EXPECT().UpdateMattermostUserStatus("user_mm_id", gomock.Any()).Times(0)

				r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)
			},
		},
		"Update status to away using legacy settings, GetConfirmation enabled and overlapping events present": {
			settings: store.Settings{
				UpdateStatus:                      true,
				ReceiveNotificationsDuringMeeting: true,
				GetConfirmation:                   true,
			},
			runAssertions: func(deps *Dependencies, client remote.Client) {
				c, papi, _, _, r := client.(*mock_remote.MockClient), deps.PluginAPI.(*mock_plugin_api.MockPluginAPI), deps.Poster.(*mock_bot.MockPoster), deps.Store.(*mock_store.MockStore), deps.Remote.(*mock_remote.MockRemote)
				moment := time.Now().UTC()
				firstBusyEvent := &remote.Event{ICalUID: "event_id-1", Start: remote.NewDateTime(moment, "UTC"), End: remote.NewDateTime(moment.Add(10*time.Minute), "UTC"), ShowAs: "busy"}
				secondBusyEvent := &remote.Event{ICalUID: "event_id-2", Start: remote.NewDateTime(moment.Add(5*time.Minute), "UTC"), End: remote.NewDateTime(moment.Add(15*time.Minute), "UTC"), ShowAs: "busy"}

				c.EXPECT().DoBatchViewCalendarRequests(gomock.Any()).Times(1).Return([]*remote.ViewCalendarResponse{
					{Events: []*remote.Event{firstBusyEvent, secondBusyEvent}, RemoteUserID: "user_remote_id"},
				}, nil)
				papi.EXPECT().GetMattermostUserStatusesByIds([]string{"user_mm_id"}).Return([]*model.Status{{Status: "online", Manual: true, UserId: "user_mm_id"}}, nil)

				papi.EXPECT().UpdateMattermostUserStatus("user_mm_id", gomock.Any()).Times(0)

				r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)
			},
		},
		"Update status to dnd using legacy settings, GetConfirmation enabled and overlapping events present": {
			settings: store.Settings{
				UpdateStatus:                      true,
				ReceiveNotificationsDuringMeeting: false,
				GetConfirmation:                   true,
			},
			runAssertions: func(deps *Dependencies, client remote.Client) {
				c, papi, _, _, r := client.(*mock_remote.MockClient), deps.PluginAPI.(*mock_plugin_api.MockPluginAPI), deps.Poster.(*mock_bot.MockPoster), deps.Store.(*mock_store.MockStore), deps.Remote.(*mock_remote.MockRemote)
				moment := time.Now().UTC()
				firstBusyEvent := &remote.Event{ICalUID: "event_id-1", Start: remote.NewDateTime(moment, "UTC"), End: remote.NewDateTime(moment.Add(10*time.Minute), "UTC"), ShowAs: "busy"}
				secondBusyEvent := &remote.Event{ICalUID: "event_id-2", Start: remote.NewDateTime(moment.Add(5*time.Minute), "UTC"), End: remote.NewDateTime(moment.Add(15*time.Minute), "UTC"), ShowAs: "busy"}

				c.EXPECT().DoBatchViewCalendarRequests(gomock.Any()).Times(1).Return([]*remote.ViewCalendarResponse{
					{Events: []*remote.Event{firstBusyEvent, secondBusyEvent}, RemoteUserID: "user_remote_id"},
				}, nil)
				papi.EXPECT().GetMattermostUserStatusesByIds([]string{"user_mm_id"}).Return([]*model.Status{{Status: "online", Manual: true, UserId: "user_mm_id"}}, nil)

				papi.EXPECT().UpdateMattermostUserStatus("user_mm_id", gomock.Any()).Times(0)

				r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			env, client := makeStatusSyncTestEnv(ctrl)

			s := env.Dependencies.Store.(*mock_store.MockStore)
			s.EXPECT().LoadUserIndex().Return(store.UserIndex{
				&store.UserShort{
					MattermostUserID: "user_mm_id",
					RemoteID:         "user_remote_id",
					Email:            "user_email@example.com",
				},
			}, nil).Times(1)
			s.EXPECT().LoadUser("user_mm_id").Return(&store.User{
				MattermostUserID: "user_mm_id",
				Remote: &remote.User{
					ID:   "user_remote_id",
					Mail: "user_email@example.com",
				},
				Settings: tc.settings,
			}, nil).Times(1)

			tc.runAssertions(env.Dependencies, client)

			mscalendar := New(env, "")
			_, _, err := mscalendar.SyncAll()
			require.Nil(t, err)
		})
	}
}

func TestSyncCustomStatusUserConfig(t *testing.T) {
	for name, tc := range map[string]struct {
		runAssertions func(deps *Dependencies, client remote.Client)
		settings      store.Settings
	}{
		"SetCustomStatus disabled": {
			settings: store.Settings{
				SetCustomStatus: false,
			},
			runAssertions: func(deps *Dependencies, client remote.Client) {
				_, _, _, _, r := client.(*mock_remote.MockClient), deps.PluginAPI.(*mock_plugin_api.MockPluginAPI), deps.Poster.(*mock_bot.MockPoster), deps.Store.(*mock_store.MockStore), deps.Remote.(*mock_remote.MockRemote)

				r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)
			},
		},
		"SetCustomStatus enabled but no event present": {
			settings: store.Settings{
				SetCustomStatus: true,
			},
			runAssertions: func(deps *Dependencies, client remote.Client) {
				c, papi, _, s, r := client.(*mock_remote.MockClient), deps.PluginAPI.(*mock_plugin_api.MockPluginAPI), deps.Poster.(*mock_bot.MockPoster), deps.Store.(*mock_store.MockStore), deps.Remote.(*mock_remote.MockRemote)

				c.EXPECT().DoBatchViewCalendarRequests(gomock.Any()).Times(1).Return([]*remote.ViewCalendarResponse{
					{Events: []*remote.Event{}, RemoteUserID: "user_remote_id"},
				}, nil)
				papi.EXPECT().GetMattermostUserStatusesByIds([]string{"user_mm_id"}).Return([]*model.Status{{Status: "online", Manual: true, UserId: "user_mm_id"}}, nil)
				papi.EXPECT().RemoveMattermostUserCustomStatus("user_mm_id").Return(nil)
				s.EXPECT().StoreUserCustomStatusUpdates("user_mm_id", false).Return(nil)

				r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)
			},
		},
		"SetCustomStatus enabled but overlapping events": {
			settings: store.Settings{
				SetCustomStatus: true,
			},
			runAssertions: func(deps *Dependencies, client remote.Client) {
				c, papi, _, s, r := client.(*mock_remote.MockClient), deps.PluginAPI.(*mock_plugin_api.MockPluginAPI), deps.Poster.(*mock_bot.MockPoster), deps.Store.(*mock_store.MockStore), deps.Remote.(*mock_remote.MockRemote)
				moment := time.Now().UTC()
				firstBusyEvent := &remote.Event{ICalUID: "event_id-1", Start: remote.NewDateTime(moment, "UTC"), End: remote.NewDateTime(moment.Add(10*time.Minute), "UTC"), ShowAs: "busy", Attendees: []*remote.Attendee{{EmailAddress: &remote.EmailAddress{Address: "mock-attendee@gmail.com"}}}}
				secondBusyEvent := &remote.Event{ICalUID: "event_id-2", Start: remote.NewDateTime(moment.Add(5*time.Minute), "UTC"), End: remote.NewDateTime(moment.Add(15*time.Minute), "UTC"), ShowAs: "busy", Attendees: []*remote.Attendee{{EmailAddress: &remote.EmailAddress{Address: "mock-attendee@gmail.com"}}}}

				c.EXPECT().DoBatchViewCalendarRequests(gomock.Any()).Times(1).Return([]*remote.ViewCalendarResponse{
					{Events: []*remote.Event{firstBusyEvent, secondBusyEvent}, RemoteUserID: "user_remote_id"},
				}, nil)
				papi.EXPECT().GetMattermostUserStatusesByIds([]string{"user_mm_id"}).Return([]*model.Status{{Status: "online", Manual: true, UserId: "user_mm_id"}}, nil)

				r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)

				papi.EXPECT().GetMattermostUser("user_mm_id").Return(&model.User{
					Id: "user_mm_id",
				}, nil)
				papi.EXPECT().UpdateMattermostUserCustomStatus("user_mm_id", &model.CustomStatus{
					Emoji:     "calendar",
					Text:      "In a meeting",
					ExpiresAt: secondBusyEvent.End.Time(),
					Duration:  "date_and_time",
				}).Return(nil)

				s.EXPECT().StoreUserCustomStatusUpdates("user_mm_id", true).Return(nil)
			},
		},
		"SetCustomStatus enabled with back-to-back events": {
			settings: store.Settings{
				SetCustomStatus: true,
			},
			runAssertions: func(deps *Dependencies, client remote.Client) {
				c, papi, _, s, r := client.(*mock_remote.MockClient), deps.PluginAPI.(*mock_plugin_api.MockPluginAPI), deps.Poster.(*mock_bot.MockPoster), deps.Store.(*mock_store.MockStore), deps.Remote.(*mock_remote.MockRemote)
				moment := time.Now().UTC()
				firstBusyEvent := &remote.Event{ICalUID: "event_id-1", Start: remote.NewDateTime(moment, "UTC"), End: remote.NewDateTime(moment.Add(10*time.Minute), "UTC"), Attendees: []*remote.Attendee{{EmailAddress: &remote.EmailAddress{Address: "mock-attendee@gmail.com"}}}, ShowAs: "busy"}
				secondBusyEvent := &remote.Event{ICalUID: "event_id-2", Start: firstBusyEvent.End, End: remote.NewDateTime(moment.Add(15*time.Minute), "UTC"), Attendees: []*remote.Attendee{{EmailAddress: &remote.EmailAddress{Address: "mock-attendee@gmail.com"}}}, ShowAs: "busy"}

				c.EXPECT().DoBatchViewCalendarRequests(gomock.Any()).Times(1).Return([]*remote.ViewCalendarResponse{
					{Events: []*remote.Event{firstBusyEvent, secondBusyEvent}, RemoteUserID: "user_remote_id"},
				}, nil)
				papi.EXPECT().GetMattermostUserStatusesByIds([]string{"user_mm_id"}).Return([]*model.Status{{Status: "online", Manual: true, UserId: "user_mm_id"}}, nil)

				r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)

				papi.EXPECT().GetMattermostUser("user_mm_id").Return(&model.User{
					Id: "user_mm_id",
				}, nil)
				papi.EXPECT().UpdateMattermostUserCustomStatus("user_mm_id", &model.CustomStatus{
					Emoji:     "calendar",
					Text:      "In a meeting",
					ExpiresAt: secondBusyEvent.End.Time(),
					Duration:  "date_and_time",
				}).Return(nil)

				s.EXPECT().StoreUserCustomStatusUpdates("user_mm_id", true).Return(nil)
			},
		},
		"SetCustomStatus enabled with non overlapping events": {
			settings: store.Settings{
				SetCustomStatus: true,
			},
			runAssertions: func(deps *Dependencies, client remote.Client) {
				c, papi, _, s, r := client.(*mock_remote.MockClient), deps.PluginAPI.(*mock_plugin_api.MockPluginAPI), deps.Poster.(*mock_bot.MockPoster), deps.Store.(*mock_store.MockStore), deps.Remote.(*mock_remote.MockRemote)
				moment := time.Now().UTC()
				firstBusyEvent := &remote.Event{ICalUID: "event_id-1", Start: remote.NewDateTime(moment, "UTC"), End: remote.NewDateTime(moment.Add(10*time.Minute), "UTC"), Attendees: []*remote.Attendee{{EmailAddress: &remote.EmailAddress{Address: "mock-attendee@gmail.com"}}}, ShowAs: "busy"}
				secondBusyEvent := &remote.Event{ICalUID: "event_id-2", Start: remote.NewDateTime(moment.Add(20*time.Minute), "UTC"), End: remote.NewDateTime(moment.Add(30*time.Minute), "UTC"), Attendees: []*remote.Attendee{{EmailAddress: &remote.EmailAddress{Address: "mock-attendee@gmail.com"}}}, ShowAs: "busy"}

				c.EXPECT().DoBatchViewCalendarRequests(gomock.Any()).Times(1).Return([]*remote.ViewCalendarResponse{
					{Events: []*remote.Event{firstBusyEvent, secondBusyEvent}, RemoteUserID: "user_remote_id"},
				}, nil)
				papi.EXPECT().GetMattermostUserStatusesByIds([]string{"user_mm_id"}).Return([]*model.Status{{Status: "online", Manual: true, UserId: "user_mm_id"}}, nil)

				r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)

				papi.EXPECT().GetMattermostUser("user_mm_id").Return(&model.User{
					Id: "user_mm_id",
				}, nil)
				papi.EXPECT().UpdateMattermostUserCustomStatus("user_mm_id", &model.CustomStatus{
					Emoji:     "calendar",
					Text:      "In a meeting",
					ExpiresAt: firstBusyEvent.End.Time(),
					Duration:  "date_and_time",
				}).Return(nil)

				s.EXPECT().StoreUserCustomStatusUpdates("user_mm_id", true).Return(nil)
			},
		},
		"SetCustomStatus enabled but event cancelled": {
			settings: store.Settings{
				SetCustomStatus: true,
			},
			runAssertions: func(deps *Dependencies, client remote.Client) {
				c, papi, _, s, r := client.(*mock_remote.MockClient), deps.PluginAPI.(*mock_plugin_api.MockPluginAPI), deps.Poster.(*mock_bot.MockPoster), deps.Store.(*mock_store.MockStore), deps.Remote.(*mock_remote.MockRemote)
				moment := time.Now().UTC()
				busyEvent := &remote.Event{ICalUID: "event_id-1", Start: remote.NewDateTime(moment, "UTC"), ShowAs: "busy", IsCancelled: true}

				c.EXPECT().DoBatchViewCalendarRequests(gomock.Any()).Times(1).Return([]*remote.ViewCalendarResponse{
					{Events: []*remote.Event{busyEvent}, RemoteUserID: "user_remote_id"},
				}, nil)
				papi.EXPECT().GetMattermostUserStatusesByIds([]string{"user_mm_id"}).Return([]*model.Status{{Status: "online", Manual: true, UserId: "user_mm_id"}}, nil)

				papi.EXPECT().RemoveMattermostUserCustomStatus("user_mm_id").Return(nil)
				s.EXPECT().StoreUserCustomStatusUpdates("user_mm_id", false).Return(nil)

				r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)
			},
		},
		"SetCustomStatus enabled but no attendee present": {
			settings: store.Settings{
				SetCustomStatus: true,
			},
			runAssertions: func(deps *Dependencies, client remote.Client) {
				c, papi, _, s, r := client.(*mock_remote.MockClient), deps.PluginAPI.(*mock_plugin_api.MockPluginAPI), deps.Poster.(*mock_bot.MockPoster), deps.Store.(*mock_store.MockStore), deps.Remote.(*mock_remote.MockRemote)
				moment := time.Now().UTC()
				busyEvent := &remote.Event{ICalUID: "event_id-1", Start: remote.NewDateTime(moment, "UTC"), ShowAs: "busy"}

				c.EXPECT().DoBatchViewCalendarRequests(gomock.Any()).Times(1).Return([]*remote.ViewCalendarResponse{
					{Events: []*remote.Event{busyEvent}, RemoteUserID: "user_remote_id"},
				}, nil)
				papi.EXPECT().GetMattermostUserStatusesByIds([]string{"user_mm_id"}).Return([]*model.Status{{Status: "online", Manual: true, UserId: "user_mm_id"}}, nil)
				papi.EXPECT().RemoveMattermostUserCustomStatus("user_mm_id").Return(nil)

				r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)

				s.EXPECT().StoreUserCustomStatusUpdates("user_mm_id", false).Return(nil)
			},
		},
		"SetCustomStatus enabled": {
			settings: store.Settings{
				SetCustomStatus: true,
			},
			runAssertions: func(deps *Dependencies, client remote.Client) {
				c, papi, _, s, r := client.(*mock_remote.MockClient), deps.PluginAPI.(*mock_plugin_api.MockPluginAPI), deps.Poster.(*mock_bot.MockPoster), deps.Store.(*mock_store.MockStore), deps.Remote.(*mock_remote.MockRemote)
				moment := time.Now().UTC()
				busyEvent := &remote.Event{ICalUID: "event_id", Start: remote.NewDateTime(moment, "UTC"), ShowAs: "busy", Attendees: []*remote.Attendee{{EmailAddress: &remote.EmailAddress{Address: "mock-attendee@gmail.com"}}}, End: remote.NewDateTime(moment.Add(10*time.Minute), "UTC")}

				c.EXPECT().DoBatchViewCalendarRequests(gomock.Any()).Times(1).Return([]*remote.ViewCalendarResponse{
					{Events: []*remote.Event{busyEvent}, RemoteUserID: "user_remote_id"},
				}, nil)
				papi.EXPECT().GetMattermostUserStatusesByIds([]string{"user_mm_id"}).Return([]*model.Status{{Status: "online", Manual: true, UserId: "user_mm_id"}}, nil)
				papi.EXPECT().GetMattermostUser("user_mm_id").Return(&model.User{
					Id: "user_mm_id",
				}, nil)
				papi.EXPECT().UpdateMattermostUserCustomStatus("user_mm_id", &model.CustomStatus{
					Emoji:     "calendar",
					Text:      "In a meeting",
					ExpiresAt: busyEvent.End.Time(),
					Duration:  "date_and_time",
				}).Return(nil)

				s.EXPECT().StoreUserCustomStatusUpdates("user_mm_id", true).Return(nil)

				r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			env, client := makeStatusSyncTestEnv(ctrl)

			s := env.Dependencies.Store.(*mock_store.MockStore)
			s.EXPECT().LoadUserIndex().Return(store.UserIndex{
				&store.UserShort{
					MattermostUserID: "user_mm_id",
					RemoteID:         "user_remote_id",
					Email:            "user_email@example.com",
				},
			}, nil).Times(1)
			s.EXPECT().LoadUser("user_mm_id").Return(&store.User{
				MattermostUserID: "user_mm_id",
				Remote: &remote.User{
					ID:   "user_remote_id",
					Mail: "user_email@example.com",
				},
				IsCustomStatusSet: true,
				Settings:          tc.settings,
			}, nil).Times(1)

			tc.runAssertions(env.Dependencies, client)

			mscalendar := New(env, "")
			_, _, err := mscalendar.SyncAll()
			require.Nil(t, err)
		})
	}
}

func TestSyncCustomStatusAndUserStatusConfig(t *testing.T) {
	for name, tc := range map[string]struct {
		runAssertions func(deps *Dependencies, client remote.Client)
		settings      store.Settings
	}{
		"Both SetCustomStatus and SetUserStatus enabled": {
			settings: store.Settings{
				SetCustomStatus:         true,
				UpdateStatusFromOptions: store.AwayStatusOption,
				GetConfirmation:         true,
			},
			runAssertions: func(deps *Dependencies, client remote.Client) {
				c, papi, poster, s, r := client.(*mock_remote.MockClient), deps.PluginAPI.(*mock_plugin_api.MockPluginAPI), deps.Poster.(*mock_bot.MockPoster), deps.Store.(*mock_store.MockStore), deps.Remote.(*mock_remote.MockRemote)
				moment := time.Now().UTC()
				busyEvent := &remote.Event{ICalUID: "event_id", Start: remote.NewDateTime(moment, "UTC"), ShowAs: "busy", Attendees: []*remote.Attendee{{EmailAddress: &remote.EmailAddress{Address: "mock-attendee@gmail.com"}}}, End: remote.NewDateTime(moment.Add(10*time.Minute), "UTC")}

				c.EXPECT().DoBatchViewCalendarRequests(gomock.Any()).Times(1).Return([]*remote.ViewCalendarResponse{
					{Events: []*remote.Event{busyEvent}, RemoteUserID: "user_remote_id"},
				}, nil)
				papi.EXPECT().GetMattermostUserStatusesByIds([]string{"user_mm_id"}).Return([]*model.Status{{Status: "online", Manual: true, UserId: "user_mm_id"}}, nil)
				papi.EXPECT().GetMattermostUser("user_mm_id").Return(&model.User{
					Id: "user_mm_id",
				}, nil)
				papi.EXPECT().UpdateMattermostUserCustomStatus("user_mm_id", &model.CustomStatus{
					Emoji:     "calendar",
					Text:      "In a meeting",
					ExpiresAt: busyEvent.End.Time(),
					Duration:  "date_and_time",
				}).Return(nil)

				s.EXPECT().StoreUserCustomStatusUpdates("user_mm_id", true).Return(nil)
				r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)

				s.EXPECT().StoreUserActiveEvents("user_mm_id", []string{"event_id " + moment.Format(time.RFC3339)})
				s.EXPECT().StoreUser(gomock.Any()).Return(nil).Times(1)
				poster.EXPECT().DMWithAttachments("user_mm_id", gomock.Any()).Times(1)
				papi.EXPECT().UpdateMattermostUserStatus("user_mm_id", gomock.Any()).Times(0)
			},
		},
		"Both SetCustomStatus and SetUserStatus enabled using legacy settings": {
			settings: store.Settings{
				SetCustomStatus:                   true,
				UpdateStatus:                      true,
				ReceiveNotificationsDuringMeeting: true,
				GetConfirmation:                   true,
			},
			runAssertions: func(deps *Dependencies, client remote.Client) {
				c, papi, poster, s, r := client.(*mock_remote.MockClient), deps.PluginAPI.(*mock_plugin_api.MockPluginAPI), deps.Poster.(*mock_bot.MockPoster), deps.Store.(*mock_store.MockStore), deps.Remote.(*mock_remote.MockRemote)
				moment := time.Now().UTC()
				busyEvent := &remote.Event{ICalUID: "event_id", Start: remote.NewDateTime(moment, "UTC"), ShowAs: "busy", Attendees: []*remote.Attendee{{EmailAddress: &remote.EmailAddress{Address: "mock-attendee@gmail.com"}}}, End: remote.NewDateTime(moment.Add(10*time.Minute), "UTC")}

				c.EXPECT().DoBatchViewCalendarRequests(gomock.Any()).Times(1).Return([]*remote.ViewCalendarResponse{
					{Events: []*remote.Event{busyEvent}, RemoteUserID: "user_remote_id"},
				}, nil)
				papi.EXPECT().GetMattermostUserStatusesByIds([]string{"user_mm_id"}).Return([]*model.Status{{Status: "online", Manual: true, UserId: "user_mm_id"}}, nil)
				papi.EXPECT().GetMattermostUser("user_mm_id").Return(&model.User{
					Id: "user_mm_id",
				}, nil)
				papi.EXPECT().UpdateMattermostUserCustomStatus("user_mm_id", &model.CustomStatus{
					Emoji:     "calendar",
					Text:      "In a meeting",
					ExpiresAt: busyEvent.End.Time(),
					Duration:  "date_and_time",
				}).Return(nil)

				s.EXPECT().StoreUserCustomStatusUpdates("user_mm_id", true).Return(nil)
				r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)

				s.EXPECT().StoreUserActiveEvents("user_mm_id", []string{"event_id " + moment.Format(time.RFC3339)})
				s.EXPECT().StoreUser(gomock.Any()).Return(nil).Times(1)
				poster.EXPECT().DMWithAttachments("user_mm_id", gomock.Any()).Times(1)
				papi.EXPECT().UpdateMattermostUserStatus("user_mm_id", gomock.Any()).Times(0)
			},
		},
		"Both SetCustomStatus and SetUserStatus disabled": {
			settings: store.Settings{},
			runAssertions: func(deps *Dependencies, client remote.Client) {
				_, _, _, _, r := client.(*mock_remote.MockClient), deps.PluginAPI.(*mock_plugin_api.MockPluginAPI), deps.Poster.(*mock_bot.MockPoster), deps.Store.(*mock_store.MockStore), deps.Remote.(*mock_remote.MockRemote)

				r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			env, client := makeStatusSyncTestEnv(ctrl)

			s := env.Dependencies.Store.(*mock_store.MockStore)
			s.EXPECT().LoadUserIndex().Return(store.UserIndex{
				&store.UserShort{
					MattermostUserID: "user_mm_id",
					RemoteID:         "user_remote_id",
					Email:            "user_email@example.com",
				},
			}, nil).Times(1)
			s.EXPECT().LoadUser("user_mm_id").Return(&store.User{
				MattermostUserID: "user_mm_id",
				Remote: &remote.User{
					ID:   "user_remote_id",
					Mail: "user_email@example.com",
				},
				IsCustomStatusSet: true,
				Settings:          tc.settings,
			}, nil).Times(1)

			tc.runAssertions(env.Dependencies, client)

			mscalendar := New(env, "")
			_, _, err := mscalendar.SyncAll()
			require.Nil(t, err)
		})
	}
}

func TestReminders(t *testing.T) {
	for name, tc := range map[string]struct {
		apiError       *remote.APIError
		remoteEvents   []*remote.Event
		eventMetadata  map[string]*store.EventMetadata
		numReminders   int
		shouldLogError bool
	}{
		"Most common case, no remote events. No reminder.": {
			remoteEvents:   []*remote.Event{},
			numReminders:   0,
			shouldLogError: false,
		},
		"One remote event, but it is too far in the future.": {
			remoteEvents: []*remote.Event{
				{ICalUID: "event_id", Start: remote.NewDateTime(time.Now().Add(20*time.Minute).UTC(), "UTC"), End: remote.NewDateTime(time.Now().Add(45*time.Minute).UTC(), "UTC")},
			},
			numReminders:   0,
			shouldLogError: false,
		},
		"One remote event, but it is in the past.": {
			remoteEvents: []*remote.Event{
				{ICalUID: "event_id", Start: remote.NewDateTime(time.Now().Add(-15*time.Minute).UTC(), "UTC"), End: remote.NewDateTime(time.Now().Add(45*time.Minute).UTC(), "UTC")},
			},
			numReminders:   0,
			shouldLogError: false,
		},
		"One remote event, but it is to soon in the future. Reminder has already occurred.": {
			remoteEvents: []*remote.Event{
				{ICalUID: "event_id", Start: remote.NewDateTime(time.Now().Add(2*time.Minute).UTC(), "UTC"), End: remote.NewDateTime(time.Now().Add(45*time.Minute).UTC(), "UTC")},
			},
			numReminders:   0,
			shouldLogError: false,
		},
		"One remote event, and is in the range for the reminder. Reminder should occur.": {
			remoteEvents: []*remote.Event{
				{ICalUID: "event_id", Start: remote.NewDateTime(time.Now().Add(7*time.Minute).UTC(), "UTC"), End: remote.NewDateTime(time.Now().Add(45*time.Minute).UTC(), "UTC")},
			},
			numReminders:   1,
			shouldLogError: false,
		},
		"Two remote event, and are in the range for the reminder. Two reminders should occur.": {
			remoteEvents: []*remote.Event{
				{ICalUID: "event_id", Start: remote.NewDateTime(time.Now().Add(7*time.Minute).UTC(), "UTC"), End: remote.NewDateTime(time.Now().Add(45*time.Minute).UTC(), "UTC")},
				{ICalUID: "event_id", Start: remote.NewDateTime(time.Now().Add(7*time.Minute).UTC(), "UTC"), End: remote.NewDateTime(time.Now().Add(45*time.Minute).UTC(), "UTC")},
			},
			numReminders:   2,
			shouldLogError: false,
		},
		"Remote event linked to channel in the range for the reminder. DM and channel reminders should occur.": {
			remoteEvents: []*remote.Event{
				{ID: "event_id_1", ICalUID: "event_id_1", Start: remote.NewDateTime(time.Now().Add(7*time.Minute).UTC(), "UTC"), End: remote.NewDateTime(time.Now().Add(45*time.Minute).UTC(), "UTC")},
			},
			eventMetadata: map[string]*store.EventMetadata{
				"event_id_1": {
					LinkedChannelIDs: map[string]struct{}{"some_channel_id": {}},
				},
			},
			numReminders:   1,
			shouldLogError: false,
		},
		"Remote recurring event linked to channel in the range for the reminder. DM and channel reminders should occur.": {
			remoteEvents: []*remote.Event{
				{ID: "event_id_1_recurring", ICalUID: "event_id_1", Start: remote.NewDateTime(time.Now().Add(7*time.Minute).UTC(), "UTC"), End: remote.NewDateTime(time.Now().Add(45*time.Minute).UTC(), "UTC")},
			},
			eventMetadata: map[string]*store.EventMetadata{
				"event_id_1": {
					LinkedChannelIDs: map[string]struct{}{"channel_id": {}},
				},
			},
			numReminders:   1,
			shouldLogError: false,
		},
		"Remote API Error. Error should be logged.": {
			remoteEvents:   []*remote.Event{},
			numReminders:   0,
			apiError:       &remote.APIError{Code: "403", Message: "Forbidden"},
			shouldLogError: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			env, client := makeStatusSyncTestEnv(ctrl)
			deps := env.Dependencies

			c, r, poster, s, logger := client.(*mock_remote.MockClient), env.Remote.(*mock_remote.MockRemote), deps.Poster.(*mock_bot.MockPoster), deps.Store.(*mock_store.MockStore), deps.Logger.(*mock_bot.MockLogger)
			s.EXPECT().LoadUserIndex().Return(store.UserIndex{
				&store.UserShort{
					MattermostUserID: "user_mm_id",
					RemoteID:         "user_remote_id",
					Email:            "user_email@example.com",
				},
			}, nil).Times(1)
			r.EXPECT().MakeSuperuserClient(context.Background()).Return(client, nil)

			loadUser := s.EXPECT().LoadUser("user_mm_id").Return(&store.User{
				MattermostUserID: "user_mm_id",
				Remote: &remote.User{
					ID:   "user_remote_id",
					Mail: "user_email@example.com",
				},
				Settings: store.Settings{ReceiveReminders: true, UpdateStatusFromOptions: store.NotSetStatusOption},
			}, nil)
			c.EXPECT().DoBatchViewCalendarRequests(gomock.Any()).Return([]*remote.ViewCalendarResponse{
				{Events: tc.remoteEvents, RemoteUserID: "user_remote_id", Error: tc.apiError},
			}, nil)

			if tc.numReminders > 0 {
				poster.EXPECT().DMWithAttachments("user_mm_id", gomock.Any()).Times(tc.numReminders)
				loadUser.Times(2)
				c.EXPECT().GetMailboxSettings("user_remote_id").Times(1).Return(&remote.MailboxSettings{TimeZone: "UTC"}, nil)

				// Metadata (linked channels test)
				for eventID, metadata := range tc.eventMetadata {
					s.EXPECT().LoadEventMetadata(eventID).Return(metadata, nil).Times(1)
					for channelID := range metadata.LinkedChannelIDs {
						poster.EXPECT().CreatePost(test.DoMatch(func(v *model.Post) bool {
							return v.ChannelId == channelID
						})).Return(nil)
					}
				}
				s.EXPECT().LoadEventMetadata(gomock.Any()).Return(nil, store.ErrNotFound).Times(tc.numReminders - len(tc.eventMetadata))
			} else {
				poster.EXPECT().DM(gomock.Any(), gomock.Any()).Times(0)
				loadUser.Times(1)
			}

			if tc.shouldLogError {
				logger.EXPECT().Warnf("Error getting availability for %s. err=%s", "user_mm_id", tc.apiError.Message).Times(1)
			} else {
				logger.EXPECT().Warnf(gomock.Any()).Times(0)
			}

			m := New(env, "")
			res, _, err := m.SyncAll()
			require.Nil(t, err)
			require.NotEmpty(t, res)
		})
	}
}

func TestRetrieveUsersToSyncIndividually(t *testing.T) {
	t.Run("no users to sync", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		env, _ := makeStatusSyncTestEnv(ctrl)

		m := New(env, "").(*mscalendar)
		jobSummary := &StatusSyncJobSummary{}

		_, _, err := m.retrieveUsersToSync([]*store.UserShort{}, jobSummary, true)
		require.ErrorIs(t, errNoUsersNeedToBeSynced, err)
	})

	t.Run("user reminders and status disabled", func(t *testing.T) {
		testUser := newTestUser()
		testUser.Settings.UpdateStatusFromOptions = store.NotSetStatusOption
		testUser.Settings.ReceiveReminders = false

		userIndex := []*store.UserShort{
			{
				MattermostUserID: testUser.MattermostUserID,
				RemoteID:         testUser.Remote.ID,
				Email:            testUser.Remote.Mail,
			},
		}

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		e, _ := makeStatusSyncTestEnv(ctrl)

		s := e.Store.(*mock_store.MockStore)
		s.EXPECT().LoadUser(testUser.MattermostUserID).Return(testUser, nil)

		m := New(e, "").(*mscalendar)
		jobSummary := &StatusSyncJobSummary{}

		_, _, err := m.retrieveUsersToSync(userIndex, jobSummary, true)
		require.ErrorIs(t, err, errNoUsersNeedToBeSynced)
	})

	t.Run("one user should be synced", func(t *testing.T) {
		testUser := newTestUser()
		testUser.Settings.UpdateStatusFromOptions = store.AwayStatusOption
		testUser.Settings.ReceiveReminders = true

		userIndex := []*store.UserShort{
			{
				MattermostUserID: testUser.MattermostUserID,
				RemoteID:         testUser.Remote.ID,
				Email:            testUser.Remote.Mail,
			},
		}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		e, client := makeStatusSyncTestEnv(ctrl)

		c, r, _, s, papi := client.(*mock_remote.MockClient), e.Remote.(*mock_remote.MockRemote), e.Poster.(*mock_bot.MockPoster), e.Store.(*mock_store.MockStore), e.PluginAPI.(*mock_plugin_api.MockPluginAPI)
		s.EXPECT().LoadUser(testUser.MattermostUserID).Return(testUser, nil).Times(2)

		events := []*remote.Event{newTestEvent("1", "", "test")}
		papi.EXPECT().GetMattermostUser(testUser.MattermostUserID)
		r.EXPECT().MakeUserClient(context.Background(), testUser.OAuth2Token, gomock.Any(), gomock.Any(), gomock.Any()).Return(client)
		c.EXPECT().GetEventsBetweenDates(testUser.Remote.ID, gomock.Any(), gomock.Any()).Return(events, nil)

		m := New(e, "").(*mscalendar)
		jobSummary := &StatusSyncJobSummary{}

		users, responses, err := m.retrieveUsersToSync(userIndex, jobSummary, true)
		require.NoError(t, err)
		require.Equal(t, []*store.User{testUser}, users)
		require.Equal(t, []*remote.ViewCalendarResponse{{
			RemoteUserID: testUser.Remote.ID,
			Events:       events,
		}}, responses)
	})

	t.Run("one user should be synced, one user shouldn't", func(t *testing.T) {
		testUser := newTestUser()
		testUser.Settings.UpdateStatusFromOptions = store.AwayStatusOption
		testUser.Settings.ReceiveReminders = true

		testUser2 := newTestUserNumbered(1)
		testUser2.Settings.UpdateStatusFromOptions = store.NotSetStatusOption

		userIndex := []*store.UserShort{
			{
				MattermostUserID: testUser.MattermostUserID,
				RemoteID:         testUser.Remote.ID,
				Email:            testUser.Remote.Mail,
			},
			{
				MattermostUserID: testUser2.MattermostUserID,
				RemoteID:         testUser2.Remote.ID,
				Email:            testUser2.Remote.Mail,
			},
		}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		e, client := makeStatusSyncTestEnv(ctrl)

		c, r, _, s, papi := client.(*mock_remote.MockClient), e.Remote.(*mock_remote.MockRemote), e.Poster.(*mock_bot.MockPoster), e.Store.(*mock_store.MockStore), e.PluginAPI.(*mock_plugin_api.MockPluginAPI)
		s.EXPECT().LoadUser(testUser.MattermostUserID).Return(testUser, nil).Times(2)
		s.EXPECT().LoadUser(testUser2.MattermostUserID).Return(testUser2, nil)

		events := []*remote.Event{newTestEvent("1", "", "test")}
		papi.EXPECT().GetMattermostUser(testUser.MattermostUserID)
		r.EXPECT().MakeUserClient(context.Background(), testUser.OAuth2Token, gomock.Any(), gomock.Any(), gomock.Any()).Return(client)
		c.EXPECT().GetEventsBetweenDates(testUser.Remote.ID, gomock.Any(), gomock.Any()).Return(events, nil)

		m := New(e, "").(*mscalendar)
		jobSummary := &StatusSyncJobSummary{}

		users, responses, err := m.retrieveUsersToSync(userIndex, jobSummary, true)
		require.NoError(t, err)
		require.Equal(t, []*store.User{testUser}, users)
		require.Equal(t, []*remote.ViewCalendarResponse{{
			RemoteUserID: testUser.Remote.ID,
			Events:       events,
		}}, responses)
	})

	t.Run("two users should be synced", func(t *testing.T) {
		testUser := newTestUserNumbered(1)
		testUser.Settings.UpdateStatusFromOptions = store.AwayStatusOption
		testUser.Settings.ReceiveReminders = true

		testUser2 := newTestUserNumbered(2)
		testUser.Settings.UpdateStatusFromOptions = store.AwayStatusOption
		testUser2.Settings.ReceiveReminders = true

		userIndex := []*store.UserShort{
			{
				MattermostUserID: testUser.MattermostUserID,
				RemoteID:         testUser.Remote.ID,
				Email:            testUser.Remote.Mail,
			},
			{
				MattermostUserID: testUser2.MattermostUserID,
				RemoteID:         testUser2.Remote.ID,
				Email:            testUser2.Remote.Mail,
			},
		}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		e, client := makeStatusSyncTestEnv(ctrl)

		c, r, _, s, papi := client.(*mock_remote.MockClient), e.Remote.(*mock_remote.MockRemote), e.Poster.(*mock_bot.MockPoster), e.Store.(*mock_store.MockStore), e.PluginAPI.(*mock_plugin_api.MockPluginAPI)
		s.EXPECT().LoadUser(testUser.MattermostUserID).Return(testUser, nil).Times(2)
		s.EXPECT().LoadUser(testUser2.MattermostUserID).Return(testUser2, nil).Times(2)

		eventsUser1 := []*remote.Event{newTestEvent("1", "", "test")}
		eventsUser2 := []*remote.Event{newTestEvent("2", "", "test2")}
		papi.EXPECT().GetMattermostUser(testUser.MattermostUserID)
		papi.EXPECT().GetMattermostUser(testUser2.MattermostUserID)
		r.EXPECT().MakeUserClient(context.Background(), testUser.OAuth2Token, gomock.Any(), gomock.Any(), gomock.Any()).Return(client)
		r.EXPECT().MakeUserClient(context.Background(), testUser2.OAuth2Token, gomock.Any(), gomock.Any(), gomock.Any()).Return(client)
		c.EXPECT().GetEventsBetweenDates(testUser.Remote.ID, gomock.Any(), gomock.Any()).Return(eventsUser1, nil)
		c.EXPECT().GetEventsBetweenDates(testUser2.Remote.ID, gomock.Any(), gomock.Any()).Return(eventsUser2, nil)

		m := New(e, "").(*mscalendar)
		jobSummary := &StatusSyncJobSummary{}

		users, responses, err := m.retrieveUsersToSync(userIndex, jobSummary, true)
		require.NoError(t, err)
		require.ElementsMatch(t, []*store.User{testUser, testUser2}, users)
		require.ElementsMatch(t, []*remote.ViewCalendarResponse{{
			RemoteUserID: testUser.Remote.ID,
			Events:       eventsUser1,
		}, {
			RemoteUserID: testUser2.Remote.ID,
			Events:       eventsUser2,
		}}, responses)
	})
}

func makeStatusSyncTestEnv(ctrl *gomock.Controller) (Env, remote.Client) {
	s := mock_store.NewMockStore(ctrl)
	poster := mock_bot.NewMockPoster(ctrl)
	mockRemote := mock_remote.NewMockRemote(ctrl)
	mockClient := mock_remote.NewMockClient(ctrl)
	mockPluginAPI := mock_plugin_api.NewMockPluginAPI(ctrl)
	logger := mock_bot.NewMockLogger(ctrl)

	env := Env{
		Config: &config.Config{},
		Dependencies: &Dependencies{
			Store:     s,
			Logger:    logger,
			Poster:    poster,
			Remote:    mockRemote,
			PluginAPI: mockPluginAPI,
		},
	}

	return env, mockClient
}

func TestGetMergedEvents(t *testing.T) {
	moment := time.Now().UTC()
	timezone := "UTC"
	for name, tc := range map[string]struct {
		events         []*remote.Event
		expectedResult []*remote.Event
	}{
		"No overlapping event or duration is greater than StatusSyncJobInterval": {
			events: []*remote.Event{
				{
					Start: remote.NewDateTime(moment, timezone),
					End:   remote.NewDateTime(moment.Add(30*time.Minute), timezone),
				},
				{
					Start: remote.NewDateTime(moment.Add(40*time.Minute), timezone),
					End:   remote.NewDateTime(moment.Add(60*time.Minute), timezone),
				},
			},
			expectedResult: []*remote.Event{
				{
					Start: remote.NewDateTime(moment, timezone),
					End:   remote.NewDateTime(moment.Add(30*time.Minute), timezone),
				},
				{
					Start: remote.NewDateTime(moment.Add(40*time.Minute), timezone),
					End:   remote.NewDateTime(moment.Add(60*time.Minute), timezone),
				},
			},
		},
		"Overlapping events": {
			events: []*remote.Event{
				{
					Start: remote.NewDateTime(moment, timezone),
					End:   remote.NewDateTime(moment.Add(30*time.Minute), timezone),
				},
				{
					Start: remote.NewDateTime(moment.Add(20*time.Minute), timezone),
					End:   remote.NewDateTime(moment.Add(60*time.Minute), timezone),
				},
			},
			expectedResult: []*remote.Event{
				{
					Start: remote.NewDateTime(moment, timezone),
					End:   remote.NewDateTime(moment.Add(60*time.Minute), timezone),
				},
			},
		},
		"No overlapping events but duration is less than StatusSyncJobInterval": {
			events: []*remote.Event{
				{
					Start: remote.NewDateTime(moment, timezone),
					End:   remote.NewDateTime(moment.Add(1*time.Minute), timezone),
				},
				{
					Start: remote.NewDateTime(moment.Add(3*time.Minute), timezone),
					End:   remote.NewDateTime(moment.Add(5*time.Minute), timezone),
				},
			},
			expectedResult: []*remote.Event{
				{
					Start: remote.NewDateTime(moment, timezone),
					End:   remote.NewDateTime(moment.Add(5*time.Minute), timezone),
				},
			},
		},
		"Overlapping events, duration is less than StatusSyncJobInterval in current event and next event": {
			events: []*remote.Event{
				{
					Start: remote.NewDateTime(moment, timezone),
					End:   remote.NewDateTime(moment.Add(3*time.Minute), timezone),
				},
				{
					Start: remote.NewDateTime(moment.Add(3*time.Minute), timezone),
					End:   remote.NewDateTime(moment.Add(5*time.Minute), timezone),
				},
			},
			expectedResult: []*remote.Event{
				{
					Start: remote.NewDateTime(moment, timezone),
					End:   remote.NewDateTime(moment.Add(5*time.Minute), timezone),
				},
			},
		},
		"No overlapping events, duration is less than StatusSyncJobInterval for current event but not for next event": {
			events: []*remote.Event{
				{
					Start: remote.NewDateTime(moment, timezone),
					End:   remote.NewDateTime(moment.Add(3*time.Minute), timezone),
				},
				{
					Start: remote.NewDateTime(moment.Add(30*time.Minute), timezone),
					End:   remote.NewDateTime(moment.Add(50*time.Minute), timezone),
				},
			},
			expectedResult: []*remote.Event{
				{
					Start: remote.NewDateTime(moment, timezone),
					End:   remote.NewDateTime(moment.Add(3*time.Minute), timezone),
				},
				{
					Start: remote.NewDateTime(moment.Add(30*time.Minute), timezone),
					End:   remote.NewDateTime(moment.Add(50*time.Minute), timezone),
				},
			},
		},
		"Overlapping events, duration is less than StatusSyncJobInterval for current event but not for next event": {
			events: []*remote.Event{
				{
					Start: remote.NewDateTime(moment, timezone),
					End:   remote.NewDateTime(moment.Add(3*time.Minute), timezone),
				},
				{
					Start: remote.NewDateTime(moment.Add(3*time.Minute), timezone),
					End:   remote.NewDateTime(moment.Add(50*time.Minute), timezone),
				},
			},
			expectedResult: []*remote.Event{
				{
					Start: remote.NewDateTime(moment, timezone),
					End:   remote.NewDateTime(moment.Add(50*time.Minute), timezone),
				},
			},
		},
		"Overlapping events, duration is less than StatusSyncJobInterval with multiple events": {
			events: []*remote.Event{
				{
					Start: remote.NewDateTime(moment, timezone),
					End:   remote.NewDateTime(moment.Add(3*time.Minute), timezone),
				},
				{
					Start: remote.NewDateTime(moment.Add(3*time.Minute), timezone),
					End:   remote.NewDateTime(moment.Add(50*time.Minute), timezone),
				},
				{
					Start: remote.NewDateTime(moment.Add(51*time.Minute), timezone),
					End:   remote.NewDateTime(moment.Add(53*time.Minute), timezone),
				},
			},
			expectedResult: []*remote.Event{
				{
					Start: remote.NewDateTime(moment, timezone),
					End:   remote.NewDateTime(moment.Add(50*time.Minute), timezone),
				},
				{
					Start: remote.NewDateTime(moment.Add(51*time.Minute), timezone),
					End:   remote.NewDateTime(moment.Add(53*time.Minute), timezone),
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			res := getMergedEvents(tc.events)
			assert.Equal(tc.expectedResult, res)
		})
	}
}
