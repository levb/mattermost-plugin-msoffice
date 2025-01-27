// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package msgraph

import (
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
)

func (c *client) DeleteCalendar(remoteUserID string, calID string) error {
	if !c.tokenHelpers.CheckUserConnected(c.mattermostUserID) {
		c.Logger.Warnf(LogUserInactive, c.mattermostUserID)
		return errors.New(ErrorUserInactive)
	}
	err := c.rbuilder.Users().ID(remoteUserID).Calendars().ID(calID).Request().Delete(c.ctx)
	if err != nil {
		c.tokenHelpers.DisconnectUserFromStoreIfNecessary(err, c.mattermostUserID)
		return errors.Wrap(err, "msgraph DeleteCalendar")
	}
	c.Logger.With(bot.LogContext{}).Infof("msgraph: DeleteCalendar deleted calendar `%v`.", calID)
	return nil
}
