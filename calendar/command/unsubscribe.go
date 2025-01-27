// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

func (c *Command) unsubscribe(_ ...string) (string, bool, error) {
	_, err := c.Engine.LoadMyEventSubscription()
	if err != nil {
		return "You are not subscribed to events.", false, nil
	}

	err = c.Engine.DeleteMyEventSubscription()
	if err != nil {
		return "", false, err
	}

	return "You have unsubscribed from events.", false, nil
}
