// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package settingspanel

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/httputils"
)

const (
	ContextIDKey          = "setting_id"
	ContextButtonValueKey = "button_value"
	ContextOptionValueKey = "selected_option"
)

type handler struct {
	panel Panel
}

func Init(h *httputils.Handler, panel Panel) {
	sh := &handler{
		panel: panel,
	}

	panelRouter := h.Router.PathPrefix("/").Subrouter()
	panelRouter.HandleFunc(panel.URL(), sh.handleAction).Methods(http.MethodPost)
}

func (sh *handler) handleAction(w http.ResponseWriter, r *http.Request) {
	mattermostUserID := r.Header.Get("Mattermost-User-ID")
	if mattermostUserID == "" {
		utils.SlackAttachmentError(w, "Error: Not authorized")
		return
	}
	var request model.PostActionIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.SlackAttachmentError(w, "Error: invalid request")
		return
	}

	id, ok := request.Context[ContextIDKey]
	if !ok {
		utils.SlackAttachmentError(w, "Error: missing setting id")
		return
	}

	value, ok := request.Context[ContextButtonValueKey]
	if !ok {
		value, ok = request.Context[ContextOptionValueKey]
		if !ok {
			utils.SlackAttachmentError(w, "Error: valid key not found")
			return
		}
	}

	idString := id.(string)
	err := sh.panel.Set(mattermostUserID, idString, value)
	if err != nil {
		utils.SlackAttachmentError(w, "Error: cannot set the property, "+err.Error())
		return
	}

	// TODO: Workaround for https://community.mattermost.com/core/pl/nphtmkowcjd8ic76tbqtapx6nc
	// See: https://mattermost.atlassian.net/browse/MM-54032
	time.Sleep(2 * time.Second)

	response := model.PostActionIntegrationResponse{}
	post, err := sh.panel.ToPost(mattermostUserID)
	if err == nil {
		response.Update = post
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		utils.SlackAttachmentError(w, "Error: unable to write response, "+err.Error())
	}
}
