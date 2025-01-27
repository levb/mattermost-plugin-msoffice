// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package flow

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/httputils"
)

type fh struct {
	flow  Flow
	store Store
}

func Init(h *httputils.Handler, flow Flow, store Store) {
	fh := &fh{
		flow:  flow,
		store: store,
	}

	flowRouter := h.Router.PathPrefix("/").Subrouter()
	flowRouter.HandleFunc(flow.URL(), fh.handleFlow).Methods(http.MethodPost)
}

func (fh *fh) handleFlow(w http.ResponseWriter, r *http.Request) {
	mattermostUserID := r.Header.Get("Mattermost-User-ID")
	if mattermostUserID == "" {
		utils.SlackAttachmentError(w, "Error: Not authorized")
		return
	}

	stepNumber, err := strconv.Atoi(r.URL.Query().Get("step"))
	if err != nil {
		utils.SlackAttachmentError(w, fmt.Sprintf("Error: Step provided is not an int, err=%s", err.Error()))
		return
	}

	step := fh.flow.Step(stepNumber)
	if step == nil {
		utils.SlackAttachmentError(w, fmt.Sprintf("Error: There is no step %d.", step))
		return
	}

	property := step.GetPropertyName()
	valueString := r.URL.Query().Get(property)
	if valueString == "" {
		utils.SlackAttachmentError(w, "Correct property not set")
		return
	}

	value := valueString == "true"
	err = fh.store.SetProperty(mattermostUserID, property, value)
	if err != nil {
		utils.SlackAttachmentError(w, "There has been a problem setting the property, err="+err.Error())
		return
	}

	response := model.PostActionIntegrationResponse{}
	post := model.Post{}
	model.ParseSlackAttachment(&post, []*model.SlackAttachment{step.ResponseSlackAttachment(value)})
	response.Update = &post

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		utils.SlackAttachmentError(w, "Error encoding response, err="+err.Error())
		return
	}

	fh.store.RemovePostID(mattermostUserID, property)

	// TODO: Workaround for https://community.mattermost.com/core/pl/nphtmkowcjd8ic76tbqtapx6nc
	// See: https://mattermost.atlassian.net/browse/MM-54032
	time.Sleep(2 * time.Second)
	fh.flow.StepDone(mattermostUserID, stepNumber, value)
}
