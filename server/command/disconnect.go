// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package command

func (c *Command) disconnect(parameters ...string) (string, bool, error) {
	err := c.Engine.DisconnectUser(c.Args.UserId)
	if err != nil {
		return "", false, err
	}
	c.Engine.ClearSettingsPosts(c.Args.UserId)

	return "Successfully disconnected your account", false, nil
}
