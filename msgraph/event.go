// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package msgraph

import (
	"net/http"
	"time"

	"github.com/pkg/errors"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
)

const (
	MicrosoftResponseStatusYes          = "accepted"
	MicrosoftResponseStatusMaybe        = "tentativelyAccepted"
	MicrosoftResponseStatusNo           = "declined"
	MicrosoftResponseStatusNone         = "none"
	MicrosoftResponseStatusOrganizer    = "organizer"
	MicrosoftResponseStatusNotResponsed = MicrosoftResponseStatusNone
)

var responseStatusConversion = map[string]string{
	MicrosoftResponseStatusYes:   remote.EventResponseStatusAccepted,
	MicrosoftResponseStatusMaybe: remote.EventResponseStatusTentative,
	MicrosoftResponseStatusNo:    remote.EventResponseStatusDeclined,
	MicrosoftResponseStatusNone:  remote.EventResponseStatusNotAnswered,
	// TODO: unused by us? Should we prefill event organizer response to this?
	MicrosoftResponseStatusOrganizer: remote.EventResponseStatusNotAnswered,
}

// converts microsoft calendar responses to our representation of fields
func normalizeEvents(events []*remote.Event) []*remote.Event {
	for i := range events {
		events[i].ResponseStatus.Response = responseStatusConversion[events[i].ResponseStatus.Response]
	}
	return events
}

func (c *client) GetEvent(remoteUserID, eventID string) (*remote.Event, error) {
	e := &remote.Event{}

	if !c.tokenHelpers.CheckUserConnected(c.mattermostUserID) {
		c.Logger.Warnf(LogUserInactive, c.mattermostUserID)
		return nil, errors.New(ErrorUserInactive)
	}

	err := c.rbuilder.Users().ID(remoteUserID).Events().ID(eventID).Request().JSONRequest(
		c.ctx, http.MethodGet, "", nil, &e)
	if err != nil {
		c.tokenHelpers.DisconnectUserFromStoreIfNecessary(err, c.mattermostUserID)
		return nil, errors.Wrap(err, "msgraph GetEvent")
	}
	return e, nil
}

func (c *client) AcceptEvent(remoteUserID, eventID string) error {
	dummy := &msgraph.EventAcceptRequestParameter{}
	if !c.tokenHelpers.CheckUserConnected(c.mattermostUserID) {
		c.Logger.Warnf(LogUserInactive, c.mattermostUserID)
		return errors.New(ErrorUserInactive)
	}
	err := c.rbuilder.Users().ID(remoteUserID).Events().ID(eventID).Accept(dummy).Request().Post(c.ctx)
	if err != nil {
		c.tokenHelpers.DisconnectUserFromStoreIfNecessary(err, c.mattermostUserID)
		return errors.Wrap(err, "msgraph Accept Event")
	}
	return nil
}

func (c *client) DeclineEvent(remoteUserID, eventID string) error {
	dummy := &msgraph.EventDeclineRequestParameter{}
	if !c.tokenHelpers.CheckUserConnected(c.mattermostUserID) {
		c.Logger.Warnf(LogUserInactive, c.mattermostUserID)
		return errors.New(ErrorUserInactive)
	}
	err := c.rbuilder.Users().ID(remoteUserID).Events().ID(eventID).Decline(dummy).Request().Post(c.ctx)
	if err != nil {
		c.tokenHelpers.DisconnectUserFromStoreIfNecessary(err, c.mattermostUserID)
		return errors.Wrap(err, "msgraph DeclineEvent")
	}
	return nil
}

func (c *client) TentativelyAcceptEvent(remoteUserID, eventID string) error {
	dummy := &msgraph.EventTentativelyAcceptRequestParameter{}
	if !c.tokenHelpers.CheckUserConnected(c.mattermostUserID) {
		c.Logger.Warnf(LogUserInactive, c.mattermostUserID)
		return errors.New(ErrorUserInactive)
	}
	err := c.rbuilder.Users().ID(remoteUserID).Events().ID(eventID).TentativelyAccept(dummy).Request().Post(c.ctx)
	if err != nil {
		c.tokenHelpers.DisconnectUserFromStoreIfNecessary(err, c.mattermostUserID)
		return errors.Wrap(err, "msgraph TentativelyAcceptEvent")
	}
	return nil
}

func (c *client) GetEventsBetweenDates(remoteUserID string, start, end time.Time) ([]*remote.Event, error) {
	paramStr := getQueryParamStringForCalendarView(start, end)
	res := &calendarViewResponse{}
	if !c.tokenHelpers.CheckUserConnected(c.mattermostUserID) {
		c.Logger.Warnf(LogUserInactive, c.mattermostUserID)
		return nil, errors.New(ErrorUserInactive)
	}
	err := c.rbuilder.Users().ID(remoteUserID).CalendarView().Request().JSONRequest(
		c.ctx, http.MethodGet, paramStr, nil, res)
	if err != nil {
		c.tokenHelpers.DisconnectUserFromStoreIfNecessary(err, c.mattermostUserID)
		return nil, errors.Wrap(err, "msgraph GetEventsBetweenDates")
	}

	return normalizeEvents(res.Value), nil
}
