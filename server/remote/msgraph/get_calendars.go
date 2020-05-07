// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package msgraph

import (
	"net/http"

	"github.com/larkox/mattermost-plugin-utils/bot/logger"
	"github.com/mattermost/mattermost-plugin-mscalendar/server/remote"
)

func (c *client) GetCalendars(remoteUserID string) ([]*remote.Calendar, error) {
	var v struct {
		Value []*remote.Calendar `json:"value"`
	}
	req := c.rbuilder.Users().ID(remoteUserID).Calendars().Request()
	req.Expand("children")
	err := req.JSONRequest(c.ctx, http.MethodGet, "", nil, &v)
	if err != nil {
		return nil, err
	}
	c.Logger.With(logger.LogContext{
		"UserID": remoteUserID,
		"v":      v.Value,
	}).Infof("msgraph: GetUserCalendars returned `%d` calendars.", len(v.Value))
	return v.Value, nil
}
