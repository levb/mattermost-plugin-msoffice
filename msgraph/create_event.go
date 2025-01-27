// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package msgraph

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
)

// CreateEvent creates a calendar event
func (c *client) CreateEvent(remoteUserID string, in *remote.Event) (*remote.Event, error) {
	var out = remote.Event{}
	if !c.tokenHelpers.CheckUserConnected(c.mattermostUserID) {
		c.Logger.Warnf(LogUserInactive, c.mattermostUserID)
		return nil, errors.New(ErrorUserInactive)
	}
	err := c.rbuilder.Users().ID(remoteUserID).Events().Request().JSONRequest(c.ctx, http.MethodPost, "", &in, &out)
	if err != nil {
		c.tokenHelpers.DisconnectUserFromStoreIfNecessary(err, c.mattermostUserID)
		return nil, errors.Wrap(err, "msgraph CreateEvent")
	}
	return &out, nil
}
