// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

func (c *Command) debugAvailability(parameters ...string) (string, bool, error) {
	switch {
	case len(parameters) == 0:
		resString, _, err := c.Engine.Sync(c.Args.UserId)
		if err != nil {
			return "", false, err
		}

		return resString, false, nil
	case len(parameters) == 1 && parameters[0] == "all":
		resString, _, err := c.Engine.SyncAll()
		if err != nil {
			return "", false, err
		}

		return resString, false, nil
	}

	return "bad syntax", false, nil
}
