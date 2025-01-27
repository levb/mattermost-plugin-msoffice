// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

import (
	"fmt"
	"net/url"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
)

func (c *Command) info(_ ...string) (string, bool, error) {
	resp := fmt.Sprintf("Mattermost %s plugin version: %s, "+
		"[%s](https://github.com/mattermost/%s/commit/%s), built %s\n",
		c.Config.Provider.DisplayName,
		c.Config.PluginVersion,
		c.Config.BuildHashShort,
		url.PathEscape(config.Provider.Repository),
		url.PathEscape(c.Config.BuildHash),
		c.Config.BuildDate)
	return resp, false, nil
}
