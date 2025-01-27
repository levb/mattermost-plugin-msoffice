// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	pluginapilicense "github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/command"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
)

// Handler handles commands
type Command struct {
	Engine    engine.Engine
	Context   *plugin.Context
	Args      *model.CommandArgs
	Config    *config.Config
	ChannelID string
}

func getNotConnectedText(pluginURL string) string {
	return fmt.Sprintf(
		"It looks like your Mattermost account is not connected to a %s account. [Click here to connect your account](%s/oauth2/connect) or use `/%s connect`.",
		config.Provider.DisplayName,
		pluginURL,
		config.Provider.CommandTrigger,
	)
}

type handleFunc func(parameters ...string) (string, bool, error)

var cmds = []*model.AutocompleteData{
	model.NewAutocompleteData("connect", "", fmt.Sprintf("Connect to your %s account", config.Provider.DisplayName)),
	model.NewAutocompleteData("disconnect", "", fmt.Sprintf("Disconnect from your %s account", config.Provider.DisplayName)),
	{ // Summary
		Trigger:  "summary",
		HelpText: "View your events for today, or edit the settings for your daily summary.",
		SubCommands: []*model.AutocompleteData{
			model.NewAutocompleteData("view", "", "View your daily summary."),
			model.NewAutocompleteData("today", "", "Display today's events."),
			model.NewAutocompleteData("tomorrow", "", "Display tomorrow's events."),
			model.NewAutocompleteData("settings", "", "View your settings for the daily summary."),
			model.NewAutocompleteData("time", "", "Set the time you would like to receive your daily summary."),
			model.NewAutocompleteData("enable", "", "Enable your daily summary."),
			model.NewAutocompleteData("disable", "", "Disable your daily summary."),
		},
	},
	model.NewAutocompleteData("viewcal", "", "View your events for the upcoming 14 days, including today."),
	{ // Create
		Trigger:  "event",
		HelpText: "Manage events.",
		SubCommands: []*model.AutocompleteData{
			model.NewAutocompleteData("create", "", "Creates a new event (desktop only)."),
		},
	},
	model.NewAutocompleteData("today", "", "Display today's events."),
	model.NewAutocompleteData("tomorrow", "", "Display tomorrow's events."),
	model.NewAutocompleteData("settings", "", "Edit your user personal settings."),
	model.NewAutocompleteData("info", "", "Read information about this version of the plugin."),
	model.NewAutocompleteData("help", "", "Read help text for the commands"),
}

// Register should be called by the plugin to register all necessary commands
func Register(client *pluginapilicense.Client) error {
	names := []string{}
	for _, subCommand := range cmds {
		names = append(names, subCommand.Trigger)
	}

	hint := "[" + strings.Join(names[:4], "|") + "...]"

	cmd := model.NewAutocompleteData(config.Provider.CommandTrigger, hint, fmt.Sprintf("Interact with your %s calendar.", config.Provider.DisplayName))
	cmd.SubCommands = cmds

	iconData, err := command.GetIconData(&client.System, fmt.Sprintf("assets/profile-%s.svg", config.Provider.Name))
	if err != nil {
		return errors.Wrap(err, "failed to get icon data")
	}

	return client.SlashCommand.Register(&model.Command{
		Trigger:              config.Provider.CommandTrigger,
		DisplayName:          config.Provider.DisplayName,
		Description:          fmt.Sprintf("Interact with your %s calendar.", config.Provider.DisplayName),
		AutoComplete:         true,
		AutoCompleteDesc:     strings.Join(names, ", "),
		AutoCompleteHint:     "(subcommand)",
		AutocompleteData:     cmd,
		AutocompleteIconData: iconData,
	})
}

// Handle should be called by the plugin when a command invocation is received from the Mattermost server.
func (c *Command) Handle() (string, bool, error) {
	cmd, parameters, err := c.isValid()
	if err != nil {
		return "", false, err
	}

	handler := c.help
	switch cmd {
	case "info":
		handler = c.info
	case "connect":
		handler = c.connect
	case "disconnect":
		handler = c.requireConnectedUser(c.disconnect)
	case "summary":
		handler = c.requireConnectedUser(c.dailySummary)
	case "viewcal":
		handler = c.requireConnectedUser(c.viewCalendar)
	case "settings":
		handler = c.requireConnectedUser(c.settings)
	case "events":
		handler = c.requireConnectedUser(c.event)
	// Admin only
	case "showcals":
		handler = c.requireConnectedUser(c.requireAdminUser(c.showCalendars))
	case "avail":
		handler = c.requireConnectedUser(c.requireAdminUser(c.debugAvailability))
	case "subscribe":
		handler = c.requireConnectedUser(c.requireAdminUser(c.subscribe))
	case "unsubscribe":
		handler = c.requireConnectedUser(c.requireAdminUser(c.unsubscribe))
	// Aliases
	case "today":
		parameters = []string{"today"}
		handler = c.requireConnectedUser(c.dailySummary)
	case "tomorrow":
		parameters = []string{"tomorrow"}
		handler = c.requireConnectedUser(c.dailySummary)
	}
	out, mustRedirectToDM, err := handler(parameters...)
	if err != nil {
		return out, false, errors.WithMessagef(err, "Command /%s %s failed", config.Provider.CommandTrigger, cmd)
	}

	return out, mustRedirectToDM, nil
}

func (c *Command) isValid() (subcommand string, parameters []string, err error) {
	if c.Context == nil || c.Args == nil {
		return "", nil, errors.New("invalid arguments to command.Handler")
	}
	split := strings.Fields(c.Args.Command)
	cmd := split[0]
	if cmd != "/"+config.Provider.CommandTrigger {
		return "", nil, fmt.Errorf("%q is not a supported command. Please contact your system administrator", cmd)
	}

	parameters = []string{}
	subcommand = ""
	if len(split) > 1 {
		subcommand = split[1]
	}
	if len(split) > 2 {
		parameters = split[2:]
	}

	return subcommand, parameters, nil
}

func (c *Command) user() *engine.User {
	return engine.NewUser(c.Args.UserId)
}

func (c *Command) requireConnectedUser(handle handleFunc) handleFunc {
	return func(parameters ...string) (string, bool, error) {
		connected, err := c.isConnected()
		if err != nil {
			return "", false, err
		}

		if !connected {
			return getNotConnectedText(c.Config.PluginURL), false, nil
		}
		return handle(parameters...)
	}
}

func (c *Command) requireAdminUser(handle handleFunc) handleFunc {
	return func(parameters ...string) (string, bool, error) {
		authorized, err := c.Engine.IsAuthorizedAdmin(c.Args.UserId)
		if err != nil {
			return "", false, err
		}
		if !authorized {
			return "Not authorized", false, nil
		}

		return handle(parameters...)
	}
}

func (c *Command) isConnected() (bool, error) {
	_, err := c.Engine.GetRemoteUser(c.Args.UserId)
	if err == store.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}
