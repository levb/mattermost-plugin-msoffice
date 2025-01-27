// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

import (
	"strings"
	"time"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine/views"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
)

func (c *Command) viewCalendar(_ ...string) (string, bool, error) {
	tz, err := c.Engine.GetTimezone(c.user())
	if err != nil {
		if strings.Contains(err.Error(), store.ErrorRefreshTokenNotSet) || strings.Contains(err.Error(), store.ErrorUserInactive) {
			return store.ErrorUserInactive, false, nil
		}

		return "Error: No timezone found", false, err
	}

	startOfCurrentDay := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	events, err := c.Engine.ViewCalendar(c.user(), startOfCurrentDay, time.Now().Add(14*24*time.Hour))
	if err != nil {
		return "", false, err
	}

	out, err := views.RenderCalendarView(events, tz)
	return out, false, err
}
