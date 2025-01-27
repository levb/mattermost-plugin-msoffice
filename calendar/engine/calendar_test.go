// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

func TestViewCalendar(t *testing.T) {
	mscalendar, mockStore, _, _, _, mockClient, _ := GetMockSetup(t)
	now := time.Now()
	from := now.Add(-time.Hour)
	to := now.Add(time.Hour)

	tests := []struct {
		name       string
		user       *User
		setupMock  func()
		assertions func(t *testing.T, events []*remote.Event, err error)
	}{
		{
			name: "error filtering with client",
			user: GetMockUser(nil, model.NewString(MockMMModelUserID), MockMMUserID, nil),
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("error loading the user")).Times(1)
			},
			assertions: func(t *testing.T, _ []*remote.Event, err error) {
				require.ErrorContains(t, err, "error loading the user")
			},
		},
		{
			name: "error getting calendar view",
			user: GetMockUser(model.NewString(MockRemoteUserID), model.NewString(MockMMModelUserID), MockMMUserID, nil),
			setupMock: func() {
				mockClient.EXPECT().GetDefaultCalendarView(MockRemoteUserID, from, to).Return(nil, fmt.Errorf("error getting calendar view")).Times(1)
			},
			assertions: func(t *testing.T, _ []*remote.Event, err error) {
				require.EqualError(t, err, "error getting calendar view")
			},
		},
		{
			name: "successful calendar view",
			user: GetMockUser(model.NewString(MockRemoteUserID), model.NewString(MockMMModelUserID), MockMMUserID, nil),
			setupMock: func() {
				mockClient.EXPECT().GetDefaultCalendarView(MockRemoteUserID, from, to).Return([]*remote.Event{{Subject: MockEventName}}, nil).Times(1)
			},
			assertions: func(t *testing.T, events []*remote.Event, err error) {
				require.NoError(t, err)
				require.NotNil(t, events)
				require.Len(t, events, 1)
				require.Equal(t, MockEventName, events[0].Subject, "Expected first event's subject to be %s, but got %s", MockEventName, events[0].Subject)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			events, err := mscalendar.ViewCalendar(tt.user, from, to)

			tt.assertions(t, events, err)
		})
	}
}

func TestGetTodayCalendarEvents(t *testing.T) {
	mscalendar, mockStore, _, _, _, mockClient, _ := GetMockSetup(t)
	now := time.Now()
	timezone := "America/Los_Angeles"
	from, to := getTodayHoursForTimezone(now, timezone)

	tests := []struct {
		name       string
		user       *User
		setupMock  func()
		assertions func(t *testing.T, events []*remote.Event, err error)
	}{
		{
			name: "error expanding remote user",
			user: GetMockUser(nil, model.NewString(MockMMModelUserID), MockMMUserID, nil),
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("error loading the user")).Times(1)
			},
			assertions: func(t *testing.T, _ []*remote.Event, err error) {
				require.ErrorContains(t, err, "error loading the user")
			},
		},
		{
			name: "error getting calendar view",
			user: GetMockUser(model.NewString(MockRemoteUserID), model.NewString(MockMMModelUserID), MockMMUserID, nil),
			setupMock: func() {
				mockClient.EXPECT().GetDefaultCalendarView(MockRemoteUserID, from, to).Return(nil, fmt.Errorf("error getting calendar view")).Times(1)
			},
			assertions: func(t *testing.T, _ []*remote.Event, err error) {
				require.EqualError(t, err, "error getting calendar view")
			},
		},
		{
			name: "successful calendar view",
			user: GetMockUser(model.NewString(MockRemoteUserID), model.NewString(MockMMModelUserID), MockMMUserID, nil),
			setupMock: func() {
				mockClient.EXPECT().GetDefaultCalendarView(MockRemoteUserID, from, to).Return([]*remote.Event{{Subject: "Today's Test Event"}}, nil).Times(1)
			},
			assertions: func(t *testing.T, events []*remote.Event, err error) {
				require.NoError(t, err)
				require.NotNil(t, events)
				require.Len(t, events, 1)
				require.Equal(t, "Today's Test Event", events[0].Subject, "Expected first event's subject to be %s, but got %s", "Today's Test Event", events[0].Subject)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			events, err := mscalendar.getTodayCalendarEvents(tt.user, now, timezone)

			tt.assertions(t, events, err)
		})
	}
}

func TestCreateCalendar(t *testing.T) {
	mscalendar, mockStore, _, _, _, mockClient, _ := GetMockSetup(t)

	tests := []struct {
		name       string
		user       *User
		calendar   *remote.Calendar
		setupMock  func()
		assertions func(t *testing.T, createdCalendar *remote.Calendar, err error)
	}{
		{
			name:     "error expanding user",
			user:     GetMockUser(nil, nil, MockMMUserID, nil),
			calendar: GetMockCalendar(MockCalendarName),
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("error loading the user")).Times(1)
			},
			assertions: func(t *testing.T, _ *remote.Calendar, err error) {
				require.ErrorContains(t, err, "error loading the user")
			},
		},
		{
			name:     "error creating calendar",
			user:     GetMockUser(model.NewString(MockRemoteUserID), model.NewString(MockMMModelUserID), MockMMUserID, nil),
			calendar: GetMockCalendar(MockCalendarName),
			setupMock: func() {
				mockClient.EXPECT().CreateCalendar(MockRemoteUserID, &remote.Calendar{Name: MockCalendarName}).Return(nil, fmt.Errorf("error creating calendar")).Times(1)
			},
			assertions: func(t *testing.T, _ *remote.Calendar, err error) {
				require.EqualError(t, err, "error creating calendar")
			},
		},
		{
			name:     "successful calendar creation",
			user:     GetMockUser(model.NewString(MockRemoteUserID), model.NewString(MockMMModelUserID), MockMMUserID, nil),
			calendar: GetMockCalendar(MockCalendarName),
			setupMock: func() {
				mockClient.EXPECT().CreateCalendar(MockRemoteUserID, &remote.Calendar{Name: MockCalendarName}).Return(&remote.Calendar{Name: "Created Test Calendar"}, nil).Times(1)
			},
			assertions: func(t *testing.T, createdCalendar *remote.Calendar, err error) {
				require.NoError(t, err)
				require.NotNil(t, createdCalendar)
				require.Equal(t, "Created Test Calendar", createdCalendar.Name, "Expected calendar name to be %s, but got %s", "Created Test Calendar", createdCalendar.Name)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			createdCalendar, err := mscalendar.CreateCalendar(tt.user, tt.calendar)
			tt.assertions(t, createdCalendar, err)
		})
	}
}

func TestCreateEvent(t *testing.T) {
	mscalendar, mockStore, mockPoster, _, mockPluginAPI, mockClient, mockLogger := GetMockSetup(t)

	tests := []struct {
		name          string
		user          *User
		event         *remote.Event
		setupMock     func()
		assertions    func(t *testing.T, createdEvent *remote.Event, err error)
		expectedEvent *remote.Event
	}{
		{
			name:  "error expanding user",
			user:  GetMockUser(nil, nil, MockMMUserID, nil),
			event: GetMockEvent(MockEventName, nil, nil, nil, nil),
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("error loading the user")).Times(1)
			},
			assertions: func(t *testing.T, createdEvent *remote.Event, err error) {
				require.ErrorContains(t, err, "error loading the user")
			},
		},
		{
			name:  "error creating direct message",
			user:  GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, nil),
			event: GetMockEvent(MockEventName, nil, nil, nil, nil),
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("not found")).Times(1)
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID)
				mockPoster.EXPECT().DM(MockMMUserID, gomock.AssignableToTypeOf(""), "testDisplayName", "testDisplayName", "testCommandTrigger").Return("", fmt.Errorf("error creating DM")).Times(1)
				mockLogger.EXPECT().Warnf("CreateEvent error creating DM. err=%v", gomock.Any())
				mockClient.EXPECT().CreateEvent(MockRemoteUserID, gomock.Any()).Return(&remote.Event{}, nil).Times(1)
			},
			assertions: func(t *testing.T, createdEvent *remote.Event, err error) {
				require.NoError(t, err)
				require.NotNil(t, createdEvent)
				require.Equal(t, &remote.Event{}, createdEvent)
			},
		},
		{
			name:  "error creating event",
			user:  GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, nil),
			event: GetMockEvent(MockEventName, nil, nil, nil, nil),
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("not found")).Times(1)
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID)
				mockPoster.EXPECT().DM(MockMMUserID, gomock.AssignableToTypeOf(""), "testDisplayName", "testDisplayName", "testCommandTrigger").Return("", fmt.Errorf("error creating DM")).Times(1).Return("", nil)
				mockClient.EXPECT().CreateEvent(MockRemoteUserID, &remote.Event{Subject: MockEventName}).Return(nil, fmt.Errorf("error creating event")).Times(1)
			},
			assertions: func(t *testing.T, createdEvent *remote.Event, err error) {
				require.EqualError(t, err, "error creating event")
			},
		},
		{
			name: "successful event creation",
			user: GetMockUser(model.NewString(MockRemoteUserID), nil, MockMMUserID, nil),
			event: GetMockEvent(
				MockEventName,
				&remote.Location{DisplayName: "Test Location"},
				&remote.DateTime{DateTime: "2024-10-01T09:00:00", TimeZone: "UTC"},
				&remote.DateTime{DateTime: "2024-10-01T10:00:00", TimeZone: "UTC"},
				[]*remote.Attendee{{EmailAddress: &remote.EmailAddress{Address: "attendee1@example.com"}}},
			),
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("not found")).Times(1)
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID)
				mockPoster.EXPECT().DM(MockMMUserID, gomock.AssignableToTypeOf(""), "testDisplayName", "testDisplayName", "testCommandTrigger").Return("", fmt.Errorf("error creating DM")).Times(1).Return("", nil)
				mockClient.EXPECT().CreateEvent(MockRemoteUserID, &remote.Event{
					Subject:   MockEventName,
					Location:  &remote.Location{DisplayName: "Test Location"},
					Start:     &remote.DateTime{DateTime: "2024-10-01T09:00:00", TimeZone: "UTC"},
					End:       &remote.DateTime{DateTime: "2024-10-01T10:00:00", TimeZone: "UTC"},
					Attendees: []*remote.Attendee{{EmailAddress: &remote.EmailAddress{Address: "attendee1@example.com"}}},
				}).Return(&remote.Event{Subject: "Created Test Event", ID: "123"}, nil).Times(1)
			},
			assertions: func(t *testing.T, createdEvent *remote.Event, err error) {
				require.NoError(t, err)
				require.NotNil(t, createdEvent)
				require.Equal(t, &remote.Event{Subject: "Created Test Event", ID: "123"}, createdEvent)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			createdEvent, err := mscalendar.CreateEvent(tt.user, tt.event, []string{MockMMUserID})
			tt.assertions(t, createdEvent, err)
		})
	}
}

func TestDeleteCalendar(t *testing.T) {
	mscalendar, mockStore, _, _, mockPluginAPI, mockClient, _ := GetMockSetup(t)
	user := GetMockUser(nil, nil, MockMMUserID, nil)

	tests := []struct {
		name       string
		setupMock  func()
		assertions func(t *testing.T, err error)
	}{
		{
			name: "error filtering with client",
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("error loading the user")).Times(1)
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error loading the user")
			},
		},
		{
			name: "error deleting calendar",
			setupMock: func() {
				user.User = &store.User{Remote: &remote.User{ID: MockRemoteUserID}}
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID)
				mockClient.EXPECT().DeleteCalendar(user.User.Remote.ID, MockCalendarID).Return(errors.New("deletion error")).Times(1)
			},
			assertions: func(t *testing.T, err error) {
				require.EqualError(t, err, "deletion error")
			},
		},
		{
			name: "successful calendar deletion",
			setupMock: func() {
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID)
				mockClient.EXPECT().DeleteCalendar(user.User.Remote.ID, MockCalendarID).Return(nil).Times(1)
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := mscalendar.DeleteCalendar(user, MockCalendarID)

			tt.assertions(t, err)
		})
	}
}

func TestFindMeetingTimes(t *testing.T) {
	mscalendar, mockStore, _, _, mockPluginAPI, mockClient, _ := GetMockSetup(t)
	user := GetMockUser(nil, nil, MockMMUserID, nil)

	meetingParams := &remote.FindMeetingTimesParameters{}

	tests := []struct {
		name       string
		setupMock  func()
		assertions func(t *testing.T, err error, results *remote.MeetingTimeSuggestionResults)
	}{
		{
			name: "error filtering with client",
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("error loading the user")).Times(1)
			},
			assertions: func(t *testing.T, err error, results *remote.MeetingTimeSuggestionResults) {
				require.ErrorContains(t, err, "error loading the user")
				require.Nil(t, results)
			},
		},
		{
			name: "error finding meeting times",
			setupMock: func() {
				user.User = &store.User{Remote: &remote.User{ID: MockRemoteUserID}}
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID)
				mockClient.EXPECT().FindMeetingTimes(user.User.Remote.ID, meetingParams).Return(nil, errors.New("finding times error")).Times(1)
			},
			assertions: func(t *testing.T, err error, results *remote.MeetingTimeSuggestionResults) {
				require.EqualError(t, err, "finding times error")
				require.Nil(t, results)
			},
		},
		{
			name: "successful meeting time retrieval",
			setupMock: func() {
				user.User = &store.User{Remote: &remote.User{ID: MockRemoteUserID}}
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID)
				mockClient.EXPECT().FindMeetingTimes(user.User.Remote.ID, meetingParams).Return(&remote.MeetingTimeSuggestionResults{}, nil).Times(1)
			},
			assertions: func(t *testing.T, err error, results *remote.MeetingTimeSuggestionResults) {
				require.NoError(t, err)
				require.NotNil(t, results)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			results, err := mscalendar.FindMeetingTimes(user, meetingParams)

			tt.assertions(t, err, results)
		})
	}
}

func TestGetCalendars(t *testing.T) {
	mscalendar, mockStore, _, _, mockPluginAPI, mockClient, _ := GetMockSetup(t)
	user := GetMockUser(nil, nil, MockMMUserID, nil)

	tests := []struct {
		name       string
		setupMock  func()
		assertions func(t *testing.T, err error, calendars []*remote.Calendar)
	}{
		{
			name: "error filtering with client",
			setupMock: func() {
				mockStore.EXPECT().LoadUser(MockMMUserID).Return(nil, errors.New("error loading the user")).Times(1)
			},
			assertions: func(t *testing.T, err error, calendars []*remote.Calendar) {
				require.ErrorContains(t, err, "error loading the user")
				require.Nil(t, calendars)
			},
		},
		{
			name: "error getting calendars",
			setupMock: func() {
				user.User = &store.User{Remote: &remote.User{ID: MockRemoteUserID}}
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID)
				mockClient.EXPECT().GetCalendars(user.User.Remote.ID).Return(nil, errors.New("getting calendars error")).Times(1)
			},
			assertions: func(t *testing.T, err error, calendars []*remote.Calendar) {
				require.EqualError(t, err, "getting calendars error")
				require.Nil(t, calendars)
			},
		},
		{
			name: "successful calendars retrieval",
			setupMock: func() {
				user.User = &store.User{Remote: &remote.User{ID: MockRemoteUserID}}
				mockPluginAPI.EXPECT().GetMattermostUser(MockMMUserID)
				mockClient.EXPECT().GetCalendars(user.User.Remote.ID).Return([]*remote.Calendar{{ID: "calendar1"}}, nil).Times(1)
			},
			assertions: func(t *testing.T, err error, calendars []*remote.Calendar) {
				require.NoError(t, err)
				require.Equal(t, []*remote.Calendar{{ID: "calendar1"}}, calendars)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			calendars, err := mscalendar.GetCalendars(user)

			tt.assertions(t, err, calendars)
		})
	}
}
