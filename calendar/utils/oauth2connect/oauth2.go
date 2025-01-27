// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package oauth2connect

import (
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/httputils"
)

type App interface {
	InitOAuth2(mattermostUserID string) (string, error)
	CompleteOAuth2(mattermostUserID, code, state string) error
}

type oa struct {
	app App

	provider config.ProviderConfig
}

func Init(h *httputils.Handler, app App, providerConfig config.ProviderConfig) {
	oa := &oa{
		app:      app,
		provider: providerConfig,
	}

	oauth2Router := h.Router.PathPrefix("/oauth2").Subrouter()
	oauth2Router.HandleFunc("/connect", oa.oauth2Connect).Methods("GET")
	oauth2Router.HandleFunc("/complete", oa.oauth2Complete).Methods("GET")
}
