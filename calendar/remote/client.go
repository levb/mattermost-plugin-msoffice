// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package remote

import (
	"net/url"
	"time"
)

type Client interface {
	Core
	Calendars
	Events
	Subscriptions
	Utils
	Unsupported
}

type Core interface {
	GetMe() (*User, error)
}

type Calendars interface {
	GetEvent(remoteUserID, eventID string) (*Event, error)
	GetCalendars(remoteUserID string) ([]*Calendar, error)
	GetDefaultCalendarView(remoteUserID string, startTime, endTime time.Time) ([]*Event, error)
	DoBatchViewCalendarRequests([]*ViewCalendarParams) ([]*ViewCalendarResponse, error)
	GetMailboxSettings(remoteUserID string) (*MailboxSettings, error)
}

type Events interface {
	CreateEvent(remoteUserID string, calendarEvent *Event) (*Event, error)
	AcceptEvent(remoteUserID, eventID string) error
	DeclineEvent(remoteUserID, eventID string) error
	TentativelyAcceptEvent(remoteUserID, eventID string) error
	GetEventsBetweenDates(remoteUserID string, start, end time.Time) ([]*Event, error)
}

type Subscriptions interface {
	CreateMySubscription(notificationURL, remoteUserID string) (*Subscription, error)
	DeleteSubscription(sub *Subscription) error
	GetNotificationData(*Notification) (*Notification, error)
	ListSubscriptions() ([]*Subscription, error)
	RenewSubscription(notificationURL, remoteUserID string, sub *Subscription) (*Subscription, error)
}

type Utils interface {
	GetSuperuserToken() (string, error)
	CallFormPost(method, path string, in url.Values, out interface{}) (responseData []byte, err error)
	CallJSON(method, path string, in, out interface{}) (responseData []byte, err error)
}

type Unsupported interface {
	CreateCalendar(remoteUserID string, calendar *Calendar) (*Calendar, error)
	DeleteCalendar(remoteUserID, calendarID string) error
	FindMeetingTimes(remoteUserID string, meetingParams *FindMeetingTimesParameters) (*MeetingTimeSuggestionResults, error)
}
