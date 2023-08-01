package command

import (
	"fmt"

	"github.com/mattermost/mattermost-plugin-mscalendar/server/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/server/store"
)

var (
	dailySummaryHelp = "### Daily summary commands:\n" +
		fmt.Sprintf("`/%s summary view` - View your daily summary\n", config.Provider.CommandTrigger) +
		fmt.Sprintf("`/%s summary settings` - View your settings for the daily summary\n", config.Provider.CommandTrigger) +
		fmt.Sprintf("`/%s summary time 8:00AM` - Set the time you would like to receive your daily summary\n", config.Provider.CommandTrigger) +
		fmt.Sprintf("`/%s summary enable` - Enable your daily summary\n", config.Provider.CommandTrigger) +
		fmt.Sprintf("`/%s summary disable` - Disable your daily summary", config.Provider.CommandTrigger)
	dailySummarySetTimeErrorMessage = fmt.Sprintf("Please enter a time, for example:\n`/%s summary time 8:00AM`", config.Provider.CommandTrigger)
)

func (c *Command) dailySummary(parameters ...string) (string, bool, error) {
	if len(parameters) == 0 {
		return dailySummaryHelp, false, nil
	}

	switch parameters[0] {
	case "view":
		postStr, err := c.MSCalendar.GetDailySummaryForUser(c.user())
		if err != nil {
			return err.Error(), false, err
		}
		return postStr, false, nil
	case "time":
		if len(parameters) != 2 {
			return dailySummarySetTimeErrorMessage, false, nil
		}
		val := parameters[1]

		dsum, err := c.MSCalendar.SetDailySummaryPostTime(c.user(), val)
		if err != nil {
			return err.Error() + "\n" + dailySummarySetTimeErrorMessage, false, nil
		}

		return dailySummaryResponse(dsum), false, nil
	case "settings":
		dsum, err := c.MSCalendar.GetDailySummarySettingsForUser(c.user())
		if err != nil {
			return err.Error() + "\nYou may need to configure your daily summary using the commands below.\n" + dailySummaryHelp, false, nil
		}

		return dailySummaryResponse(dsum), false, nil
	case "enable":
		dsum, err := c.MSCalendar.SetDailySummaryEnabled(c.user(), true)
		if err != nil {
			return err.Error(), false, err
		}

		return dailySummaryResponse(dsum), false, nil
	case "disable":
		dsum, err := c.MSCalendar.SetDailySummaryEnabled(c.user(), false)
		if err != nil {
			return err.Error(), false, err
		}
		return dailySummaryResponse(dsum), false, nil
	default:
		return "Invalid command. Please try again\n\n" + dailySummaryHelp, false, nil
	}
}

func dailySummaryResponse(dsum *store.DailySummaryUserSettings) string {
	if dsum.PostTime == "" {
		return "Your daily summary time is not yet configured.\n" + dailySummarySetTimeErrorMessage
	}

	enableStr := ""
	if !dsum.Enable {
		enableStr = fmt.Sprintf(", but is disabled. Enable it with `/%s summary enable`", config.Provider.CommandTrigger)
	}
	return fmt.Sprintf("Your daily summary is configured to show at %s %s%s.", dsum.PostTime, dsum.Timezone, enableStr)
}
