// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
)

func getDailySummaryHelp() string {
	return "### Daily summary commands:\n" +
		fmt.Sprintf("`/%s summary view` - View your daily summary\n", config.Provider.CommandTrigger) +
		fmt.Sprintf("`/%s summary settings` - View your settings for the daily summary\n", config.Provider.CommandTrigger) +
		fmt.Sprintf("`/%s summary time 8:00AM` - Set the time you would like to receive your daily summary\n", config.Provider.CommandTrigger) +
		fmt.Sprintf("`/%s summary enable` - Enable your daily summary\n", config.Provider.CommandTrigger) +
		fmt.Sprintf("`/%s summary disable` - Disable your daily summary", config.Provider.CommandTrigger)
}

func getDailySummarySetTimeErrorMessage() string {
	return fmt.Sprintf("Please enter a time, for example:\n`/%s summary time 8:00AM`", config.Provider.CommandTrigger)
}

func (c *Command) dailySummary(parameters ...string) (string, bool, error) {
	if len(parameters) == 0 {
		return getDailySummaryHelp(), false, nil
	}

	switch parameters[0] {
	case "view", "today":
		postStr, err := c.Engine.GetDaySummaryForUser(time.Now(), c.user())
		if err != nil {
			if strings.Contains(err.Error(), store.ErrorRefreshTokenNotSet) || strings.Contains(err.Error(), store.ErrorUserInactive) {
				return store.ErrorUserInactive, false, nil
			}

			return err.Error(), false, err
		}
		return postStr, false, nil
	case "tomorrow":
		postStr, err := c.Engine.GetDaySummaryForUser(time.Now().Add(time.Hour*24), c.user())
		if err != nil {
			if strings.Contains(err.Error(), store.ErrorRefreshTokenNotSet) || strings.Contains(err.Error(), store.ErrorUserInactive) {
				return store.ErrorUserInactive, false, nil
			}

			return err.Error(), false, err
		}
		return postStr, false, nil
	case "time":
		if len(parameters) != 2 {
			return getDailySummarySetTimeErrorMessage(), false, nil
		}
		val := parameters[1]

		dsum, err := c.Engine.SetDailySummaryPostTime(c.user(), val)
		if err != nil {
			if strings.Contains(err.Error(), store.ErrorRefreshTokenNotSet) || strings.Contains(err.Error(), store.ErrorUserInactive) {
				return store.ErrorUserInactive, false, nil
			}

			return err.Error() + "\n" + getDailySummarySetTimeErrorMessage(), false, nil
		}

		return dailySummaryResponse(dsum), false, nil
	case "settings":
		dsum, err := c.Engine.GetDailySummarySettingsForUser(c.user())
		if err != nil {
			if strings.Contains(err.Error(), store.ErrorRefreshTokenNotSet) || strings.Contains(err.Error(), store.ErrorUserInactive) {
				return store.ErrorUserInactive, false, nil
			}

			return err.Error() + "\nYou may need to configure your daily summary using the commands below.\n" + getDailySummaryHelp(), false, nil
		}

		return dailySummaryResponse(dsum), false, nil
	case "enable":
		dsum, err := c.Engine.SetDailySummaryEnabled(c.user(), true)
		if err != nil {
			return err.Error(), false, err
		}

		return dailySummaryResponse(dsum), false, nil
	case "disable":
		dsum, err := c.Engine.SetDailySummaryEnabled(c.user(), false)
		if err != nil {
			return err.Error(), false, err
		}
		return dailySummaryResponse(dsum), false, nil
	}
	return "Invalid command. Please try again\n\n" + getDailySummaryHelp(), false, nil
}

func dailySummaryResponse(dsum *store.DailySummaryUserSettings) string {
	if dsum.PostTime == "" {
		return "Your daily summary time is not yet configured.\n" + getDailySummarySetTimeErrorMessage()
	}

	enableStr := ""
	if !dsum.Enable {
		enableStr = fmt.Sprintf(", but is disabled. Enable it with `/%s summary enable`", config.Provider.CommandTrigger)
	}
	return fmt.Sprintf("Your daily summary is configured to show at %s %s%s.", dsum.PostTime, dsum.Timezone, enableStr)
}
