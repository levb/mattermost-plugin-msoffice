// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package oauth2connect

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-mscalendar/server/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/server/utils/httputils"
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
	oauth2Router.HandleFunc("/complete", func(w http.ResponseWriter, r *http.Request) {
		oa.oauth2Complete(w, r, oa.provider.DisplayName)
	}).Methods("GET")
}
