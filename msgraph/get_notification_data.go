// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package msgraph

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
)

func (c *client) GetNotificationData(orig *remote.Notification) (*remote.Notification, error) {
	n := *orig
	wh := n.Webhook.(*webhook)
	if !c.tokenHelpers.CheckUserConnected(c.mattermostUserID) {
		c.Logger.Warnf(LogUserInactive, c.mattermostUserID)
		return nil, errors.New(ErrorUserInactive)
	}
	switch wh.ResourceData.DataType {
	case "#Microsoft.Graph.Event":
		event := remote.Event{}
		_, err := c.CallJSON(http.MethodGet, wh.Resource, nil, &event)
		if err != nil {
			c.tokenHelpers.DisconnectUserFromStoreIfNecessary(err, c.mattermostUserID)
			c.Logger.With(bot.LogContext{
				"Resource":       wh.Resource,
				"subscriptionID": wh.SubscriptionID,
			}).Infof("msgraph: failed to fetch notification data resource: `%v`.", err)
			return nil, errors.Wrap(err, "msgraph GetNotificationData")
		}
		n.Event = &event
		n.ChangeType = wh.ChangeType
		n.IsBare = false

	default:
		c.Logger.With(bot.LogContext{
			"subscriptionID": wh.SubscriptionID,
		}).Infof("msgraph: unknown resource type: `%s`.", wh.ResourceData.DataType)
		return nil, errors.New("unknown resource type: " + wh.ResourceData.DataType)
	}

	return &n, nil
}
