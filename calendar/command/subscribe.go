// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

import (
	"fmt"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils"
)

func (c *Command) subscribe(parameters ...string) (string, bool, error) {
	if len(parameters) > 0 && parameters[0] == "list" {
		return c.debugList()
	}

	_, err := c.Engine.LoadMyEventSubscription()
	if err == nil {
		return "You are already subscribed to events.", false, nil
	}

	_, err = c.Engine.CreateMyEventSubscription()
	if err != nil {
		return "", false, err
	}
	return "You are now subscribed to events.", false, nil
}

func (c *Command) debugList() (string, bool, error) {
	subs, err := c.Engine.ListRemoteSubscriptions()
	if err != nil {
		return "", false, err
	}
	return fmt.Sprintf("Subscriptions:%s", utils.JSONBlock(subs)), false, nil
}
