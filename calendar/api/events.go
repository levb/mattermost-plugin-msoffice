// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine/views"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/httputils"
)

const (
	createEventDateTimeFormat = "2006-01-02 15:04"
	createEventDateFormat     = "2006-01-02"
)

type createEventPayload struct {
	AllDay    bool     `json:"all_day"`
	Attendees []string `json:"attendees"`
	Date      string   `json:"date"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
	// Reminder  bool     `json:"reminder"
	Description string `json:"description,omitempty"`
	Subject     string `json:"subject"`
	Location    string `json:"location,omitempty"`
	ChannelID   string `json:"channel_id"`
}

func (cep createEventPayload) ToRemoteEvent(loc *time.Location) (*remote.Event, error) {
	var evt remote.Event

	evt.IsAllDay = cep.AllDay

	start, err := cep.parseStartTime(loc)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing start time")
	}

	end, err := cep.parseEndTime(loc)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing start time")
	}

	if !cep.AllDay {
		evt.Start = &remote.DateTime{
			DateTime: start.Format(remote.RFC3339NanoNoTimezone),
			TimeZone: loc.String(),
		}
		evt.End = &remote.DateTime{
			DateTime: end.Format(remote.RFC3339NanoNoTimezone),
			TimeZone: loc.String(),
		}
	} else {
		date, err := cep.parseDate(loc)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing date")
		}

		evt.Start = &remote.DateTime{
			DateTime: time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc).Format(remote.RFC3339NanoNoTimezone),
			TimeZone: loc.String(),
		}
		evt.End = &remote.DateTime{
			DateTime: time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 99, loc).Format(remote.RFC3339NanoNoTimezone),
			TimeZone: loc.String(),
		}
	}

	if cep.Description != "" {
		evt.Body = &remote.ItemBody{
			Content:     cep.Description,
			ContentType: "text/plain",
		}
	}
	evt.Subject = cep.Subject
	if cep.Location != "" {
		evt.Location = &remote.Location{
			DisplayName: cep.Location,
		}
	}

	return &evt, nil
}

func (cep createEventPayload) parseStartTime(loc *time.Location) (time.Time, error) {
	return time.ParseInLocation(createEventDateTimeFormat, fmt.Sprintf("%s %s", cep.Date, cep.StartTime), loc)
}

func (cep createEventPayload) parseEndTime(loc *time.Location) (time.Time, error) {
	return time.ParseInLocation(createEventDateTimeFormat, fmt.Sprintf("%s %s", cep.Date, cep.EndTime), loc)
}

func (cep createEventPayload) parseDate(loc *time.Location) (time.Time, error) {
	return time.ParseInLocation(createEventDateFormat, cep.Date, loc)
}

func (cep createEventPayload) IsValid(loc *time.Location) error {
	if cep.Subject == "" {
		return fmt.Errorf("subject must not be empty")
	}

	if cep.Date == "" {
		return fmt.Errorf("date must not be empty")
	}

	_, err := cep.parseDate(loc)
	if err != nil {
		return fmt.Errorf("invalid date")
	}

	if cep.StartTime == "" && cep.EndTime == "" && !cep.AllDay {
		return fmt.Errorf("start time/end time must be set or event should last all day")
	}

	start, err := cep.parseStartTime(loc)
	if err != nil {
		return fmt.Errorf("please use a valid start time")
	}

	if start.Before(time.Now()) {
		return fmt.Errorf("please select a start date and time that is not prior to the current time")
	}

	end, err := cep.parseEndTime(loc)
	if err != nil {
		return fmt.Errorf("please use a valid end time")
	}

	if end.Before(time.Now()) {
		return fmt.Errorf("please select an end date and time that is not prior to the current time")
	}

	if start.After(end) {
		return fmt.Errorf("end date cannot be earlier than start date")
	}

	return nil
}

func (api *api) createEvent(w http.ResponseWriter, r *http.Request) {
	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	if mattermostUserID == "" {
		api.Logger.Errorf("createEvent, unauthorized user")
		httputils.WriteUnauthorizedError(w, fmt.Errorf("unauthorized"))
		return
	}

	user, errStore := api.Store.LoadUser(mattermostUserID)
	if errStore != nil && !errors.Is(errStore, store.ErrNotFound) {
		api.Logger.With(bot.LogContext{"err": errStore}).Errorf("createEvent, error occurred while loading user from store")
		httputils.WriteInternalServerError(w, errStore)
		return
	}
	if errors.Is(errStore, store.ErrNotFound) {
		api.Logger.With(bot.LogContext{"err": errStore.Error()}).Errorf("createEvent, user not found in store")
		httputils.WriteUnauthorizedError(w, fmt.Errorf("unauthorized"))
		return
	}

	var payload createEventPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		api.Logger.With(bot.LogContext{"err": err.Error()}).Errorf("createEvent, error occurred while decoding event payload")
		httputils.WriteBadRequestError(w, err)
		return
	}
	defer r.Body.Close()

	if payload.ChannelID != "" {
		if !api.PluginAPI.CanLinkEventToChannel(payload.ChannelID, user.MattermostUserID) {
			api.Logger.With(bot.LogContext{"userID": mattermostUserID, "channelID": payload.ChannelID}).Errorf("createEvent, user don't have permission to link events in the selected channel")
			httputils.WriteBadRequestError(w, fmt.Errorf("you don't have permission to link events in the selected channel"))
			return
		}
	}

	client := api.Remote.MakeUserClient(context.Background(), user.OAuth2Token, mattermostUserID, api.Poster, api.Store)

	mailbox, errMailbox := client.GetMailboxSettings(user.Remote.ID)
	if errMailbox != nil {
		api.Logger.With(bot.LogContext{"err": errMailbox.Error(), "userID": mattermostUserID}).Errorf("createEvent, error occurred while getting mailbox settings for user")
		httputils.WriteInternalServerError(w, errMailbox)
		return
	}

	loc, errLocation := time.LoadLocation(mailbox.TimeZone)
	if errLocation != nil {
		api.Logger.With(bot.LogContext{"err": errLocation.Error(), "timezone": mailbox.TimeZone}).Errorf("createEvent, error occurred while loading mailbox timezone location")
		httputils.WriteInternalServerError(w, errLocation)
		return
	}

	if err := payload.IsValid(loc); err != nil {
		api.Logger.Errorf("createEvent, invalid payload")
		httputils.WriteBadRequestError(w, err)
		return
	}

	event, errParse := payload.ToRemoteEvent(loc)
	if errParse != nil {
		api.Logger.With(bot.LogContext{"err": errParse.Error()}).Errorf("createEvent, error occurred while creating remote event from payload")
		httputils.WriteBadRequestError(w, errParse)
		return
	}

	for _, pa := range payload.Attendees {
		var emailAddress string

		if strings.Contains(pa, "@") {
			emailAddress = pa
		} else {
			attendeeUser, err := api.Store.LoadUser(pa)
			if err != nil {
				api.Logger.With(bot.LogContext{"err": err.Error(), "attendee_mm_id": pa}).Errorf("error loading attendee from mattermost user id")
				continue
			}

			emailAddress = attendeeUser.Remote.Mail
		}

		event.Attendees = append(event.Attendees, &remote.Attendee{
			EmailAddress: &remote.EmailAddress{
				Address: emailAddress,
			},
		})
	}

	event, err := client.CreateEvent(user.Remote.ID, event)
	if err != nil {
		api.Logger.With(bot.LogContext{"err": err.Error()}).Errorf("createEvent, error occurred while creating event")
		httputils.WriteInternalServerError(w, err)
		return
	}

	attachment, err := views.RenderEventAsAttachment(event, mailbox.TimeZone, views.ShowTimezoneOption(mailbox.TimeZone))
	if err != nil {
		api.Logger.With(bot.LogContext{"err": err.Error()}).Errorf("createEvent, error rendering event as attachment")
	}

	// Event linking
	if payload.ChannelID != "" {
		if err := api.Store.StoreUserLinkedEvent(user.MattermostUserID, event.ICalUID, payload.ChannelID); err != nil {
			api.Poster.DM(mattermostUserID, "Your event **%s** could not be linked to a channel. Please contact an administrator for more details.", event.Subject)
			api.Logger.With(bot.LogContext{"err": err.Error(), "userID": user.MattermostUserID}).Errorf("createEvent, error occurred while storing user linked event")
			httputils.WriteInternalServerError(w, err)
			return
		}

		if err := api.Store.AddLinkedChannelToEvent(event.ICalUID, payload.ChannelID); err != nil {
			api.Logger.With(bot.LogContext{"err": err}).Errorf("error linking event to channel")
			defer func() {
				api.Poster.DM(mattermostUserID, "You event **%s** could not be linked to a channel. Please contact an administrator for more details.", event.Subject)
			}()
		} else {
			post := &model.Post{
				Message:   fmt.Sprintf("The event **%s** was linked to this channel by @%s", event.Subject, user.MattermostUsername),
				ChannelId: payload.ChannelID,
			}
			if attachment != nil {
				model.ParseSlackAttachment(post, []*model.SlackAttachment{attachment})
			}
			if err := api.Poster.CreatePost(post); err != nil {
				api.Logger.With(bot.LogContext{"err": err}).Errorf("error sending post to channel about linked event")
			}
		}
	} else {
		if attachment == nil {
			api.Poster.DM(mattermostUserID, "Your event: **%s** was created successfully.", event.Subject)
		} else {
			api.Poster.DMWithMessageAndAttachments(mattermostUserID, "Your event was created successfully.", attachment)
		}
	}

	httputils.WriteJSONResponse(w, `{"ok": true}`, http.StatusCreated)
}
