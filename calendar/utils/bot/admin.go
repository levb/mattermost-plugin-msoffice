// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package bot

import (
	"strings"
)

type Admin interface {
	IsUserAdmin(mattermostUserID string) bool
}

func (bot *bot) IsUserAdmin(mattermostUserID string) bool {
	list := strings.Split(bot.AdminUserIDs, ",")
	for _, u := range list {
		if mattermostUserID == strings.TrimSpace(u) {
			return true
		}
	}
	return false
}
