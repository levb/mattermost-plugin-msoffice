// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package msgraph

import (
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
)

const (
	ErrorUserInactive        = "You have been marked inactive because your refresh token is expired. Please disconnect and reconnect your account again."
	LogUserInactive          = "User %s is inactive. Please disconnect and reconnect your account."
	ErrorRefreshTokenExpired = "The refresh token has expired due to inactivity"
)

func (c *client) GetMe() (*remote.User, error) {
	graphUser, err := c.rbuilder.Me().Request().Get(c.ctx)
	if err != nil {
		return nil, errors.Wrap(err, "msgraph GetMe")
	}

	if graphUser.ID == nil {
		return nil, errors.New("user has no ID")
	}
	if graphUser.DisplayName == nil {
		return nil, errors.New("user has no Display Name")
	}
	if graphUser.UserPrincipalName == nil {
		return nil, errors.New("user has no Principal Name")
	}
	if graphUser.Mail == nil {
		return nil, errors.New("user has no email address. Make sure the Microsoft account is associated to an Outlook product")
	}

	user := &remote.User{
		ID:                *graphUser.ID,
		DisplayName:       *graphUser.DisplayName,
		UserPrincipalName: *graphUser.UserPrincipalName,
		Mail:              *graphUser.Mail,
	}

	return user, nil
}
