// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

func (c *Command) event(parameters ...string) (string, bool, error) {
	if len(parameters) == 0 {
		return getDailySummaryHelp(), false, nil
	}

	if parameters[0] == "create" {
		return "Creating events is only supported on desktop.", false, nil
	}

	return "", false, nil
}
