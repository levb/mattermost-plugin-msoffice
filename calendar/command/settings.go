// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

func (c *Command) settings(_ ...string) (string, bool, error) {
	c.Engine.PrintSettings(c.Args.UserId)
	return "", true, nil
}
