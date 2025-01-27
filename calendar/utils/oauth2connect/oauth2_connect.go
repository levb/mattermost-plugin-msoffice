// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package oauth2connect

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/httputils"
)

func (oa *oa) oauth2Connect(w http.ResponseWriter, r *http.Request) {
	mattermostUserID := r.Header.Get("Mattermost-User-ID")
	if mattermostUserID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	redirectURL, err := oa.app.InitOAuth2(mattermostUserID)
	if err != nil {
		httputils.WriteInternalServerError(w, err)
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
