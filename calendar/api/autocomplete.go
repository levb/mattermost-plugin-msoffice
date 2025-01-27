// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/httputils"
)

func (api *api) autocompleteConnectedUsers(w http.ResponseWriter, r *http.Request) {
	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	_, err := api.Store.LoadUser(mattermostUserID)
	if mattermostUserID == "" || errors.Is(err, store.ErrNotFound) {
		api.Logger.With(bot.LogContext{"err": err.Error()}).Errorf("user unauthorized")
		httputils.WriteUnauthorizedError(w, fmt.Errorf("unauthorized"))
		return
	}

	searchString := r.URL.Query().Get("search")
	results, err := api.Store.SearchInUserIndex(searchString, 10)
	if err != nil {
		api.Logger.With(bot.LogContext{"err": err.Error()}).Errorf("unable to search in user index")
		utils.SlackAttachmentError(w, "unable to search in user index: "+err.Error())
		httputils.WriteInternalServerError(w, err)
		return
	}

	if err := httputils.WriteJSONResponse(w, results.ToDTO(), http.StatusOK); err != nil {
		api.Logger.With(bot.LogContext{"err": err.Error()}).Errorf("error sending response to user")
		httputils.WriteInternalServerError(w, err)
	}
}
