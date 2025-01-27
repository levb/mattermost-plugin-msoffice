// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/httputils"
)

func (api *api) notification(w http.ResponseWriter, req *http.Request) {
	if api.NotificationProcessor != nil {
		err := api.NotificationProcessor.Enqueue(api.Env.Remote.HandleWebhook(w, req)...)
		if err != nil {
			api.Logger.With(bot.LogContext{"err": err.Error()}).Errorf("notification, error occurred while adding webhook event to notification queue")
			httputils.WriteInternalServerError(w, err)
			return
		}
	}
}
