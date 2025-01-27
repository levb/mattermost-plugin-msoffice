// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package msgraph

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
)

func (c *client) GetCalendars(remoteUserID string) ([]*remote.Calendar, error) {
	var v struct {
		Value []*remote.Calendar `json:"value"`
	}

	if !c.tokenHelpers.CheckUserConnected(c.mattermostUserID) {
		c.Logger.Warnf(LogUserInactive, c.mattermostUserID)
		return nil, errors.New(ErrorUserInactive)
	}

	req := c.rbuilder.Users().ID(remoteUserID).Calendars().Request()
	req.Expand("children")
	err := req.JSONRequest(c.ctx, http.MethodGet, "", nil, &v)
	if err != nil {
		c.tokenHelpers.DisconnectUserFromStoreIfNecessary(err, c.mattermostUserID)
		return nil, errors.Wrap(err, "msgraph GetCalendars")
	}
	c.Logger.With(bot.LogContext{
		"UserID": remoteUserID,
		"v":      v.Value,
	}).Infof("msgraph: GetUserCalendars returned `%d` calendars.", len(v.Value))
	return v.Value, nil
}
