// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

import (
	"fmt"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
)

const (
	ConnectBotAlreadyConnectedTemplate = "The bot account is already connected to %s account `%s`. To connect to a different account, first run `/%s disconnect_bot`."
	ConnectBotSuccessTemplate          = "[Click here to link the bot's %s account.](%s/oauth2/connect_bot)"
	ConnectAlreadyConnectedTemplate    = "Your Mattermost account is already connected to %s account `%s`. To connect to a different account, first run `/%s disconnect`."
	ConnectErrorMessage                = "There has been a problem while trying to connect. err="
)

func (c *Command) connect(_ ...string) (string, bool, error) {
	ru, err := c.Engine.GetRemoteUser(c.Args.UserId)
	if err == nil {
		return fmt.Sprintf(ConnectAlreadyConnectedTemplate, config.Provider.DisplayName, ru.Mail, config.Provider.CommandTrigger), false, nil
	}

	out := ""

	err = c.Engine.Welcome(c.Args.UserId)
	if err != nil {
		out = ConnectErrorMessage + err.Error()
	}

	return out, true, nil
}
