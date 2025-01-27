// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package msgraph

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
)

// FindMeetingTimes finds meeting time suggestions for a calendar event
func (c *client) FindMeetingTimes(remoteUserID string, params *remote.FindMeetingTimesParameters) (*remote.MeetingTimeSuggestionResults, error) {
	meetingsOut := &remote.MeetingTimeSuggestionResults{}
	if !c.tokenHelpers.CheckUserConnected(c.mattermostUserID) {
		c.Logger.Warnf(LogUserInactive, c.mattermostUserID)
		return nil, errors.New(ErrorUserInactive)
	}
	req := c.rbuilder.Users().ID(remoteUserID).FindMeetingTimes(nil).Request()
	err := req.JSONRequest(c.ctx, http.MethodPost, "", &params, &meetingsOut)
	if err != nil {
		c.tokenHelpers.DisconnectUserFromStoreIfNecessary(err, c.mattermostUserID)
		return nil, errors.Wrap(err, "msgraph FindMeetingTimes")
	}
	return meetingsOut, nil
}
