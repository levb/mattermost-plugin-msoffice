// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

import (
	"strings"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils"
)

func (c *Command) showCalendars(_ ...string) (string, bool, error) {
	resp, err := c.Engine.GetCalendars(c.user())
	if err != nil {
		if strings.Contains(err.Error(), store.ErrorRefreshTokenNotSet) || strings.Contains(err.Error(), store.ErrorUserInactive) {
			return store.ErrorUserInactive, false, nil
		}

		return "", false, err
	}
	return utils.JSONBlock(resp), false, nil
}
