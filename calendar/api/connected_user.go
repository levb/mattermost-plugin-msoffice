// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/httputils"
)

func (api *api) connectedUserHandler(w http.ResponseWriter, r *http.Request) {
	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	if mattermostUserID == "" {
		api.Logger.Errorf("connectedUserHandler, unauthorized user")
		httputils.WriteUnauthorizedError(w, fmt.Errorf("unauthorized"))
		return
	}

	_, errStore := api.Store.LoadUser(mattermostUserID)
	if errStore != nil && !errors.Is(errStore, store.ErrNotFound) {
		api.Logger.With(bot.LogContext{"err": errStore.Error()}).Errorf("connectedUserHandler, error occurred while loading user from store")
		httputils.WriteInternalServerError(w, errStore)
		return
	}
	if errors.Is(errStore, store.ErrNotFound) {
		api.Logger.With(bot.LogContext{"err": errStore.Error()}).Errorf("connectedUserHandler, user not found in store")
		httputils.WriteUnauthorizedError(w, fmt.Errorf("unauthorized"))
		return
	}

	w.Write([]byte(`{"is_connected": true}`))
}
