// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine/views"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils"
)

func (api *api) preprocessAction(w http.ResponseWriter, req *http.Request) (mscal engine.Engine, user *engine.User, eventID string, option string, postID string) {
	mattermostUserID := req.Header.Get("Mattermost-User-ID")
	if mattermostUserID == "" {
		utils.SlackAttachmentError(w, "Error: not authorized")
		return nil, nil, "", "", ""
	}

	var request model.PostActionIntegrationRequest
	if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
		utils.SlackAttachmentError(w, "Error: invalid request")
		return nil, nil, "", "", ""
	}

	eventID, ok := request.Context[config.EventIDKey].(string)
	if !ok {
		utils.SlackAttachmentError(w, "Error: missing event ID")
		return nil, nil, "", "", ""
	}
	option, _ = request.Context["selected_option"].(string)
	mscal = engine.New(api.Env, mattermostUserID)

	return mscal, engine.NewUser(mattermostUserID), eventID, option, request.PostId
}

func (api *api) postActionAccept(w http.ResponseWriter, req *http.Request) {
	localEngine, user, eventID, _, _ := api.preprocessAction(w, req)
	if eventID == "" {
		return
	}
	err := localEngine.AcceptEvent(user, eventID)
	if err != nil {
		api.Logger.Warnf("Failed to accept event. err=%v", err)
		utils.SlackAttachmentError(w, "Error: Failed to accept event: "+err.Error())
		return
	}
}

func (api *api) postActionDecline(w http.ResponseWriter, req *http.Request) {
	localEngine, user, eventID, _, _ := api.preprocessAction(w, req)
	if eventID == "" {
		return
	}
	err := localEngine.DeclineEvent(user, eventID)
	if err != nil {
		utils.SlackAttachmentError(w, "Error: Failed to decline event: "+err.Error())
		return
	}
}

func (api *api) postActionTentative(w http.ResponseWriter, req *http.Request) {
	localEngine, user, eventID, _, _ := api.preprocessAction(w, req)
	if eventID == "" {
		return
	}
	err := localEngine.TentativelyAcceptEvent(user, eventID)
	if err != nil {
		utils.SlackAttachmentError(w, "Error: Failed to tentatively accept event: "+err.Error())
		return
	}
}

func (api *api) postActionRespond(w http.ResponseWriter, req *http.Request) {
	calendar, user, eventID, option, postID := api.preprocessAction(w, req)
	if eventID == "" {
		return
	}
	err := calendar.RespondToEvent(user, eventID, option)
	if err != nil && !isAcceptedError(err) && !isNotFoundError(err) && !isCanceledError(err) {
		utils.SlackAttachmentError(w, "Error: Failed to respond to event: "+err.Error())
		return
	}

	if err != nil && isCanceledError(err) {
		utils.SlackAttachmentError(w, "Error: Cannot respond to the event because it is already canceled.")
		return
	}

	p, appErr := api.PluginAPI.GetPost(postID)
	if appErr != nil {
		utils.SlackAttachmentError(w, "Error: Failed to update the post: "+appErr.Error())
		return
	}

	sas := p.Attachments()
	if len(sas) == 0 {
		utils.SlackAttachmentError(w, "Error: Failed to update the post: No attachments found")
		return
	}

	sa := sas[0]

	if err == nil || isAcceptedError(err) {
		sa.Fields = append(sa.Fields, &model.SlackAttachmentField{
			Title: "Response",
			Value: fmt.Sprintf("You have %s this event", prettyOption(option)),
			Short: false,
		})
	}

	sa.Actions = []*model.PostAction{}
	postResponse := model.PostActionIntegrationResponse{}
	model.ParseSlackAttachment(p, []*model.SlackAttachment{sa})

	postResponse.Update = p

	if err != nil && isNotFoundError(err) {
		postResponse.EphemeralText = "Event has changed since this message. Please change your status directly on MS Calendar."
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(postResponse); err != nil {
		utils.SlackAttachmentError(w, "Error: unable to write response, "+err.Error())
	}
}

func prettyOption(option string) string {
	switch option {
	case engine.OptionYes:
		return "accepted"
	case engine.OptionNo:
		return "declined"
	case engine.OptionMaybe:
		return "tentatively accepted"
	default:
		return ""
	}
}

func (api *api) postActionConfirmStatusChange(w http.ResponseWriter, req *http.Request) {
	mattermostUserID := req.Header.Get("Mattermost-User-ID")
	if mattermostUserID == "" {
		utils.SlackAttachmentError(w, "Not authorized.")
		return
	}

	response := model.PostActionIntegrationResponse{}
	post := &model.Post{}

	var request model.PostActionIntegrationRequest
	if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
		utils.SlackAttachmentError(w, "Invalid request.")
		return
	}

	value, ok := request.Context["value"].(bool)
	if !ok {
		utils.SlackAttachmentError(w, `No recognizable value for property "value".`)
		return
	}

	returnText := "The status has not been changed."
	if value {
		changeTo, ok := request.Context["change_to"]
		if !ok {
			utils.SlackAttachmentError(w, "No state to change to.")
			return
		}
		stringChangeTo := changeTo.(string)
		prettyChangeTo, ok := request.Context["pretty_change_to"]
		if !ok {
			prettyChangeTo = changeTo
		}
		stringPrettyChangeTo := prettyChangeTo.(string)

		status, err := api.PluginAPI.GetMattermostUserStatus(mattermostUserID)
		if err != nil {
			utils.SlackAttachmentError(w, "Cannot get current status.")
			api.Logger.Debugf("cannot get user status, err=%s", err)
			return
		}

		user, err := api.Store.LoadUser(mattermostUserID)
		if err != nil {
			utils.SlackAttachmentError(w, "Cannot load user")
			return
		}

		user.LastStatus = ""
		if status.Manual {
			user.LastStatus = status.Status
		}

		err = api.Store.StoreUser(user)
		if err != nil {
			utils.SlackAttachmentError(w, "Cannot update user")
		}
		api.PluginAPI.UpdateMattermostUserStatus(mattermostUserID, stringChangeTo)
		returnText = fmt.Sprintf("The status has been changed to %s.", stringPrettyChangeTo)
	}

	eventInfo, err := getEventInfo(request.Context)
	if err != nil {
		utils.SlackAttachmentError(w, err.Error())
		return
	}

	if eventInfo != "" {
		returnText = eventInfo + "\n" + returnText
	}

	sa := &model.SlackAttachment{
		Title:    "Status Change",
		Text:     returnText,
		Fallback: "Status Change: " + returnText,
	}

	model.ParseSlackAttachment(post, []*model.SlackAttachment{sa})

	response.Update = post
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		utils.SlackAttachmentError(w, "Error: unable to write response, "+err.Error())
	}
}

func getEventInfo(ctx map[string]interface{}) (string, error) {
	hasEvent, ok := ctx["hasEvent"].(bool)
	if !ok {
		return "", errors.New("cannot check whether there is an event attached")
	}
	if !hasEvent {
		return "", nil
	}

	subject, ok := ctx["subject"].(string)
	if !ok {
		return "", errors.New("cannot find the event subject")
	}

	weblink, ok := ctx["weblink"].(string)
	if !ok {
		return "", errors.New("cannot find the event weblink")
	}

	marshalledStartTime, ok := ctx["startTime"].(string)
	if !ok {
		return "", errors.New("cannot find the event start time")
	}
	var startTime time.Time
	json.Unmarshal([]byte(marshalledStartTime), &startTime)

	return views.RenderEventWillStartLine(subject, weblink, startTime), nil
}

func isAcceptedError(err error) bool {
	return strings.Contains(err.Error(), "202 Accepted")
}

func isNotFoundError(err error) bool {
	return strings.Contains(err.Error(), "404 Not Found")
}

func isCanceledError(err error) bool {
	return strings.Contains(err.Error(), "You can't respond to a meeting that's been canceled.")
}
