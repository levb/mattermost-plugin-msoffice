// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package views

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
)

var prettyStatuses = map[string]string{
	model.StatusOnline:  "Online",
	model.StatusAway:    "Away",
	model.StatusDnd:     "Do Not Disturb",
	model.StatusOffline: "Offline",
}

func RenderStatusChangeNotificationView(events []*remote.Event, status, url string) *model.SlackAttachment {
	for _, e := range events {
		if e.Start.Time().After(time.Now()) {
			return statusChangeAttachments(e, status, url)
		}
	}

	nEvents := len(events)
	if nEvents > 0 && status == model.StatusDnd {
		return statusChangeAttachments(events[nEvents-1], status, url)
	}

	return statusChangeAttachments(nil, status, url)
}

func RenderEventWillStartLine(subject, weblink string, startTime time.Time) string {
	link, _ := url.QueryUnescape(weblink)
	eventString := fmt.Sprintf("Your event [%s](%s) will start soon.", subject, link)
	if subject == "" {
		eventString = fmt.Sprintf("[An event with no subject](%s) will start soon.", link)
	}
	if startTime.Before(time.Now()) {
		eventString = fmt.Sprintf("Your event [%s](%s) is ongoing.", subject, link)
		if subject == "" {
			eventString = fmt.Sprintf("[An event with no subject](%s) is ongoing.", link)
		}
	}
	return eventString
}

func renderScheduleItem(event *remote.Event, status string) string {
	if event == nil {
		return fmt.Sprintf("You have no upcoming events.\n Shall I change your status back to %s?", prettyStatuses[status])
	}

	resp := RenderEventWillStartLine(event.Subject, event.Weblink, event.Start.Time())

	resp += fmt.Sprintf("\nShall I change your status to %s?", prettyStatuses[status])
	return resp
}

func statusChangeAttachments(event *remote.Event, status, url string) *model.SlackAttachment {
	actionYes := &model.PostAction{
		Name: "Yes",
		Integration: &model.PostActionIntegration{
			URL: url,
			Context: map[string]interface{}{
				"value":            true,
				"change_to":        status,
				"pretty_change_to": prettyStatuses[status],
				"hasEvent":         false,
			},
		},
	}

	actionNo := &model.PostAction{
		Name: "No",
		Integration: &model.PostActionIntegration{
			URL: url,
			Context: map[string]interface{}{
				"value":    false,
				"hasEvent": false,
			},
		},
	}

	if event != nil {
		marshalledStart, _ := json.Marshal(event.Start.Time())
		actionYes.Integration.Context["hasEvent"] = true
		actionYes.Integration.Context["subject"] = event.Subject
		actionYes.Integration.Context["weblink"] = event.Weblink
		actionYes.Integration.Context["startTime"] = string(marshalledStart)

		actionNo.Integration.Context["hasEvent"] = true
		actionNo.Integration.Context["subject"] = event.Subject
		actionNo.Integration.Context["weblink"] = event.Weblink
		actionNo.Integration.Context["startTime"] = string(marshalledStart)
	}

	title := "Status change"
	text := renderScheduleItem(event, status)
	sa := &model.SlackAttachment{
		Title:    title,
		Text:     text,
		Actions:  []*model.PostAction{actionYes, actionNo},
		Fallback: fmt.Sprintf("%s: %s", title, text),
	}

	return sa
}
