// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"fmt"
	"sort"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine/views"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
)

const (
	calendarViewTimeWindowSize    = 10 * time.Minute
	StatusSyncJobInterval         = 5 * time.Minute
	upcomingEventNotificationTime = 10 * time.Minute

	upcomingEventNotificationWindow = (StatusSyncJobInterval * 11) / 10 // 110% of the interval
	logTruncateMsg                  = "We've truncated the logs due to too many messages"
	logTruncateLimit                = 5
)

var (
	errNoUsersNeedToBeSynced = errors.New("no users need to be synced")
)

type StatusSyncJobSummary struct {
	NumberOfUsersFailedStatusChanged int
	NumberOfUsersStatusChanged       int
	NumberOfUsersProcessed           int
}

type Availability interface {
	GetCalendarViews(users []*store.User) ([]*remote.ViewCalendarResponse, error)
	Sync(mattermostUserID string) (string, *StatusSyncJobSummary, error)
	SyncAll() (string, *StatusSyncJobSummary, error)
}

func (m *mscalendar) Sync(mattermostUserID string) (string, *StatusSyncJobSummary, error) {
	user, err := m.Store.LoadUserFromIndex(mattermostUserID)
	if err != nil {
		return "", nil, err
	}

	userIndex := store.UserIndex{user}

	err = m.Filter(withSuperuserClient)
	if err != nil && !errors.Is(err, remote.ErrSuperUserClientNotSupported) {
		return "", &StatusSyncJobSummary{}, errors.Wrap(err, "not able to filter the super user client")
	}

	return m.syncUsers(userIndex, errors.Is(err, remote.ErrSuperUserClientNotSupported))
}

func (m *mscalendar) SyncAll() (string, *StatusSyncJobSummary, error) {
	userIndex, err := m.Store.LoadUserIndex()
	if err != nil {
		if err.Error() == "not found" {
			return "No users found in user index", &StatusSyncJobSummary{}, nil
		}
		return "", &StatusSyncJobSummary{}, errors.Wrap(err, "not able to load the users from user index")
	}

	err = m.Filter(withSuperuserClient)
	if err != nil && !errors.Is(err, remote.ErrSuperUserClientNotSupported) {
		return "", &StatusSyncJobSummary{}, errors.Wrap(err, "not able to filter the super user client")
	}

	result, jobSummary, err := m.syncUsers(userIndex, errors.Is(err, remote.ErrSuperUserClientNotSupported))
	if result != "" && err != nil {
		return result, jobSummary, nil
	}

	return result, jobSummary, err
}

// retrieveUsersToSync retrieves the users and their calendar data to sync up and send notifications
// The parameter fetchIndividually determines if the calendar data should be fetched while we loop the
// users (using individual credentials) or on a batch after the loop.
func (m *mscalendar) retrieveUsersToSync(userIndex store.UserIndex, syncJobSummary *StatusSyncJobSummary, fetchIndividually bool) ([]*store.User, []*remote.ViewCalendarResponse, error) {
	start := time.Now().UTC()
	end := time.Now().UTC().Add(calendarViewTimeWindowSize)

	numberOfLogs := 0
	users := []*store.User{}
	calendarViews := []*remote.ViewCalendarResponse{}
	for _, u := range userIndex {
		// TODO fetch users from kvstore in batches, and process in batches instead of all at once
		user, err := m.Store.LoadUser(u.MattermostUserID)
		if err != nil {
			syncJobSummary.NumberOfUsersFailedStatusChanged++
			if numberOfLogs < logTruncateLimit {
				m.Logger.Warnf("Not able to load user %s from user index. err=%v", u.MattermostUserID, err)
			} else if numberOfLogs == logTruncateLimit {
				m.Logger.Warnf(logTruncateMsg)
			}
			numberOfLogs++

			// In case of error in loading, skip this user and continue with the next user
			continue
		}

		// If user does not have the proper features enabled, just go to the next one
		if !(user.IsConfiguredForStatusUpdates() || user.IsConfiguredForCustomStatusUpdates() || user.Settings.ReceiveReminders) {
			continue
		}

		if fetchIndividually {
			engine, err := m.FilterCopy(withActingUser(user.MattermostUserID))
			if err != nil {
				m.Logger.Warnf("Not able to enable active user %s from user index. err=%v", user.MattermostUserID, err)
				continue
			}

			calendarUser := newUserFromStoredUser(user)
			calendarEvents, err := engine.GetCalendarEvents(calendarUser, start, end, true)
			if err != nil {
				syncJobSummary.NumberOfUsersFailedStatusChanged++
				m.Logger.With(bot.LogContext{
					"user": u.MattermostUserID,
					"err":  err,
				}).Warnf("could not get calendar events")
				continue
			}

			calendarViews = append(calendarViews, calendarEvents)
		}

		users = append(users, user)
	}

	if len(users) == 0 {
		return users, calendarViews, errNoUsersNeedToBeSynced
	}

	if !fetchIndividually {
		var err error
		calendarViews, err = m.GetCalendarViews(users)
		if err != nil {
			return users, calendarViews, errors.Wrap(err, "not able to get calendar views for connected users")
		}
	}

	if len(calendarViews) == 0 {
		return users, calendarViews, fmt.Errorf("no calendar views found")
	}

	// Sort events for all fetched calendar views
	for _, view := range calendarViews {
		events := view.Events
		sort.Slice(events, func(i, j int) bool {
			return events[i].Start.Time().UnixMicro() < events[j].Start.Time().UnixMicro()
		})
	}

	return users, calendarViews, nil
}

func (m *mscalendar) syncUsers(userIndex store.UserIndex, fetchIndividually bool) (string, *StatusSyncJobSummary, error) {
	syncJobSummary := &StatusSyncJobSummary{}
	if len(userIndex) == 0 {
		return "No connected users found", syncJobSummary, nil
	}
	syncJobSummary.NumberOfUsersProcessed = len(userIndex)

	users, calendarViews, err := m.retrieveUsersToSync(userIndex, syncJobSummary, fetchIndividually)
	if err != nil {
		return err.Error(), syncJobSummary, errors.Wrapf(err, "error retrieving users to sync (individually=%v)", fetchIndividually)
	}

	m.deliverReminders(users, calendarViews, fetchIndividually)
	out, numberOfUsersStatusChanged, numberOfUsersFailedStatusChanged, err := m.setUserStatuses(users, calendarViews)
	if err != nil {
		return "", syncJobSummary, errors.Wrap(err, "error setting the user statuses")
	}

	syncJobSummary.NumberOfUsersFailedStatusChanged += numberOfUsersFailedStatusChanged
	syncJobSummary.NumberOfUsersStatusChanged = numberOfUsersStatusChanged

	return out, syncJobSummary, nil
}

func (m *mscalendar) deliverReminders(users []*store.User, calendarViews []*remote.ViewCalendarResponse, fetchIndividually bool) {
	numberOfLogs := 0
	toNotify := []*store.User{}
	for _, u := range users {
		if u.Settings.ReceiveReminders {
			toNotify = append(toNotify, u)
		}
	}
	if len(toNotify) == 0 {
		return
	}

	usersByRemoteID := map[string]*store.User{}
	for _, u := range toNotify {
		usersByRemoteID[u.Remote.ID] = u
	}

	for _, view := range calendarViews {
		user, ok := usersByRemoteID[view.RemoteUserID]
		if !ok {
			continue
		}
		if view.Error != nil {
			if numberOfLogs < logTruncateLimit {
				m.Logger.Warnf("Error getting availability for %s. err=%s", user.MattermostUserID, view.Error.Message)
			} else if numberOfLogs == logTruncateLimit {
				m.Logger.Warnf(logTruncateMsg)
			}
			numberOfLogs++
			continue
		}

		mattermostUserID := usersByRemoteID[view.RemoteUserID].MattermostUserID
		if fetchIndividually {
			engine, err := m.FilterCopy(withActingUser(user.MattermostUserID))
			if err != nil {
				m.Logger.With(bot.LogContext{"err": err}).Errorf("error getting engine for user")
				continue
			}
			engine.notifyUpcomingEvents(mattermostUserID, view.Events)
		} else {
			m.notifyUpcomingEvents(mattermostUserID, view.Events)
		}
	}
}

func (m *mscalendar) setUserStatuses(users []*store.User, calendarViews []*remote.ViewCalendarResponse) (string, int, int, error) {
	numberOfLogs, numberOfUserStatusChange, numberOfUserErrorInStatusChange := 0, 0, 0
	toUpdate := []*store.User{}
	for _, u := range users {
		if u.IsConfiguredForStatusUpdates() || u.IsConfiguredForCustomStatusUpdates() {
			toUpdate = append(toUpdate, u)
		}
	}
	if len(toUpdate) == 0 {
		return "No users want their status updated", numberOfUserStatusChange, numberOfUserErrorInStatusChange, nil
	}

	mattermostUserIDs := []string{}
	usersByRemoteID := map[string]*store.User{}
	for _, u := range toUpdate {
		mattermostUserIDs = append(mattermostUserIDs, u.MattermostUserID)
		usersByRemoteID[u.Remote.ID] = u
	}

	statuses, appErr := m.PluginAPI.GetMattermostUserStatusesByIds(mattermostUserIDs)
	if appErr != nil {
		return "", numberOfUserStatusChange, numberOfUserErrorInStatusChange, errors.Wrap(appErr, "error in getting Mattermost user statuses for connected users")
	}
	statusMap := map[string]*model.Status{}
	for _, s := range statuses {
		statusMap[s.UserId] = s
	}

	var res string
	for _, view := range calendarViews {
		isStatusChanged := false
		user, ok := usersByRemoteID[view.RemoteUserID]
		if !ok {
			continue
		}
		if view.Error != nil {
			if numberOfLogs < logTruncateLimit {
				m.Logger.Warnf("Error getting availability for %s. err=%s", user.MattermostUserID, view.Error.Message)
			} else if numberOfLogs == logTruncateLimit {
				m.Logger.Warnf(logTruncateMsg)
			}
			numberOfLogs++
			numberOfUserErrorInStatusChange++
			continue
		}

		mattermostUserID := usersByRemoteID[view.RemoteUserID].MattermostUserID
		status, ok := statusMap[mattermostUserID]
		if !ok {
			continue
		}

		events := filterBusyAndAttendeeEvents(view.Events)
		events = getMergedEvents(events)

		var err error
		if user.IsConfiguredForStatusUpdates() {
			res, isStatusChanged, err = m.setStatusFromCalendarView(user, status, events)
			if err != nil {
				if numberOfLogs < logTruncateLimit {
					m.Logger.Warnf("Error setting user %s status. err=%v", user.MattermostUserID, err)
				} else if numberOfLogs == logTruncateLimit {
					m.Logger.Warnf(logTruncateMsg)
				}
				numberOfLogs++
				numberOfUserErrorInStatusChange++
			}
			if isStatusChanged {
				numberOfUserStatusChange++
			}
		}

		if user.IsConfiguredForCustomStatusUpdates() {
			res, isStatusChanged, err = m.setCustomStatusFromCalendarView(user, events)
			if err != nil {
				if numberOfLogs < logTruncateLimit {
					m.Logger.Warnf("Error setting user %s custom status. err=%v", user.MattermostUserID, err)
				} else if numberOfLogs == logTruncateLimit {
					m.Logger.Warnf(logTruncateMsg)
				}
				numberOfLogs++
				numberOfUserErrorInStatusChange++
			}

			// Increment count only when we have not updated the status of the user from the options to have status change count per user.
			if isStatusChanged && user.Settings.UpdateStatusFromOptions == store.NotSetStatusOption {
				numberOfUserStatusChange++
			}
		}
	}

	if res != "" {
		return res, numberOfUserStatusChange, numberOfUserErrorInStatusChange, nil
	}

	return utils.JSONBlock(calendarViews), numberOfUserStatusChange, numberOfUserErrorInStatusChange, nil
}

func (m *mscalendar) setCustomStatusFromCalendarView(user *store.User, events []*remote.Event) (string, bool, error) {
	isStatusChanged := false
	if !user.IsConfiguredForCustomStatusUpdates() {
		return "User doesn't want to set custom status", isStatusChanged, nil
	}

	if len(events) == 0 {
		if user.IsCustomStatusSet {
			if err := m.PluginAPI.RemoveMattermostUserCustomStatus(user.MattermostUserID); err != nil {
				m.Logger.Warnf("Error removing user %s custom status. err=%v", user.MattermostUserID, err)
			}

			if err := m.Store.StoreUserCustomStatusUpdates(user.MattermostUserID, false); err != nil {
				return "", isStatusChanged, err
			}
		}

		return "No event present to set custom status", isStatusChanged, nil
	}

	currentUser, err := m.PluginAPI.GetMattermostUser(user.MattermostUserID)
	if err != nil {
		return "", isStatusChanged, err
	}

	currentCustomStatus := currentUser.GetCustomStatus()
	if currentCustomStatus != nil && !user.IsCustomStatusSet {
		return "User already has a custom status set, ignoring custom status change", isStatusChanged, nil
	}

	if appErr := m.PluginAPI.UpdateMattermostUserCustomStatus(user.MattermostUserID, &model.CustomStatus{
		Emoji:     "calendar",
		Text:      "In a meeting",
		ExpiresAt: events[0].End.Time(),
		Duration:  "date_and_time",
	}); appErr != nil {
		return "", isStatusChanged, appErr
	}

	isStatusChanged = true
	if err := m.Store.StoreUserCustomStatusUpdates(user.MattermostUserID, true); err != nil {
		return "", isStatusChanged, err
	}

	return "", isStatusChanged, nil
}

func (m *mscalendar) setStatusFromCalendarView(user *store.User, status *model.Status, events []*remote.Event) (string, bool, error) {
	isStatusChanged := false
	currentStatus := status.Status
	if !user.IsConfiguredForStatusUpdates() {
		return "No value set from options to update status", isStatusChanged, nil
	}

	if currentStatus == model.StatusOffline && !user.Settings.GetConfirmation {
		return "User offline and does not want status change confirmations. No status change", isStatusChanged, nil
	}

	busyStatus := model.StatusDnd
	if user.Settings.UpdateStatusFromOptions == store.AwayStatusOption {
		busyStatus = model.StatusAway
	}

	if len(user.ActiveEvents) == 0 && len(events) == 0 {
		return "No events in local or remote. No status change.", isStatusChanged, nil
	}

	if len(user.ActiveEvents) > 0 && len(events) == 0 {
		message := fmt.Sprintf("User is no longer busy in calendar, but is not set to busy (%s). No status change.", busyStatus)
		if currentStatus == busyStatus {
			message = "User is no longer busy in calendar. Set status to online."
			if user.LastStatus != "" {
				message = fmt.Sprintf("User is no longer busy in calendar. Set status to previous status (%s)", user.LastStatus)
			}
			err := m.setStatusOrAskUser(user, status, events, true)
			if err != nil {
				return "", isStatusChanged, errors.Wrapf(err, "error in setting user status for user %s", user.MattermostUserID)
			}
			isStatusChanged = true
		}

		err := m.Store.StoreUserActiveEvents(user.MattermostUserID, []string{})
		if err != nil {
			return "", isStatusChanged, errors.Wrapf(err, "error in storing active events for user %s", user.MattermostUserID)
		}
		return message, isStatusChanged, nil
	}

	remoteHashes := []string{}
	for _, e := range events {
		if e.IsCancelled {
			continue
		}
		h := fmt.Sprintf("%s %s", e.ICalUID, e.Start.Time().UTC().Format(time.RFC3339))
		remoteHashes = append(remoteHashes, h)
	}

	if len(user.ActiveEvents) == 0 {
		var err error
		if currentStatus == busyStatus {
			user.LastStatus = ""
			if status.Manual {
				user.LastStatus = currentStatus
			}
			m.Store.StoreUser(user)
			err = m.Store.StoreUserActiveEvents(user.MattermostUserID, remoteHashes)
			if err != nil {
				return "", isStatusChanged, errors.Wrapf(err, "error in storing active events for user %s", user.MattermostUserID)
			}
			return "User was already marked as busy. No status change.", isStatusChanged, nil
		}
		err = m.setStatusOrAskUser(user, status, events, false)
		if err != nil {
			return "", isStatusChanged, errors.Wrapf(err, "error in setting user status for user %s", user.MattermostUserID)
		}
		isStatusChanged = true
		err = m.Store.StoreUserActiveEvents(user.MattermostUserID, remoteHashes)
		if err != nil {
			return "", isStatusChanged, errors.Wrapf(err, "error in storing active events for user %s", user.MattermostUserID)
		}
		return fmt.Sprintf("User was free, but is now busy (%s). Set status to busy.", busyStatus), isStatusChanged, nil
	}

	newEventExists := false
	for _, r := range remoteHashes {
		found := false
		for _, loc := range user.ActiveEvents {
			if loc == r {
				found = true
				break
			}
		}
		if !found {
			newEventExists = true
			break
		}
	}

	if !newEventExists {
		return fmt.Sprintf("No change in active events. Total number of events: %d", len(events)), isStatusChanged, nil
	}

	message := "User is already busy. No status change."
	if currentStatus != busyStatus {
		err := m.setStatusOrAskUser(user, status, events, false)
		if err != nil {
			return "", isStatusChanged, errors.Wrapf(err, "error in setting user status for user %s", user.MattermostUserID)
		}
		isStatusChanged = true
		message = fmt.Sprintf("User was free, but is now busy. Set status to busy (%s).", busyStatus)
	}

	err := m.Store.StoreUserActiveEvents(user.MattermostUserID, remoteHashes)
	if err != nil {
		return "", isStatusChanged, errors.Wrapf(err, "error in storing active events for user %s", user.MattermostUserID)
	}

	return message, isStatusChanged, nil
}

// setStatusOrAskUser to which status change, and whether it should update the status automatically or ask the user.
// - user: the user to change the status. We use user.LastStatus to determine the status the user had before the beginning of the meeting.
// - currentStatus: currentStatus, to decide whether to store this status when the user is free. This gets assigned to user.LastStatus at the beginning of the meeting.
// - events: the list of events that are triggering this status change
// - isFree: whether the user is free or busy, to decide to which status to change
func (m *mscalendar) setStatusOrAskUser(user *store.User, currentStatus *model.Status, events []*remote.Event, isFree bool) error {
	toSet := model.StatusOnline
	if isFree && user.LastStatus != "" {
		toSet = user.LastStatus
		user.LastStatus = ""
	}

	if !isFree {
		toSet = model.StatusDnd
		if user.Settings.UpdateStatusFromOptions == store.AwayStatusOption {
			toSet = model.StatusAway
		}
		if !user.Settings.GetConfirmation {
			user.LastStatus = ""
			if currentStatus.Manual {
				user.LastStatus = currentStatus.Status
			}
		}
	}

	err := m.Store.StoreUser(user)
	if err != nil {
		return err
	}

	if !user.Settings.GetConfirmation {
		_, appErr := m.PluginAPI.UpdateMattermostUserStatus(user.MattermostUserID, toSet)
		if appErr != nil {
			return appErr
		}
		return nil
	}

	url := fmt.Sprintf("%s%s%s", m.Config.PluginURLPath, config.PathPostAction, config.PathConfirmStatusChange)
	_, err = m.Poster.DMWithAttachments(user.MattermostUserID, views.RenderStatusChangeNotificationView(events, toSet, url))
	if err != nil {
		return err
	}
	return nil
}

func (m *mscalendar) GetCalendarEvents(user *User, start, end time.Time, excludeDeclined bool) (*remote.ViewCalendarResponse, error) {
	err := m.Filter(withClient)
	if err != nil {
		return nil, errors.Wrap(err, "error withClient in GetCalendarEvents")
	}

	events, err := m.client.GetEventsBetweenDates(user.Remote.ID, start, end)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting events for user %s", user.MattermostUserID)
	}

	if excludeDeclined {
		events = m.excludeDeclinedEvents(events)
	}

	return &remote.ViewCalendarResponse{
		RemoteUserID: user.Remote.ID,
		Events:       events,
	}, nil
}

func (m *mscalendar) GetCalendarViews(users []*store.User) ([]*remote.ViewCalendarResponse, error) {
	err := m.Filter(withClient)
	if err != nil {
		return nil, fmt.Errorf("error withClient in GetCalendarViews: %w", err)
	}

	start := time.Now().UTC()
	end := time.Now().UTC().Add(calendarViewTimeWindowSize)

	params := []*remote.ViewCalendarParams{}
	for _, u := range users {
		params = append(params, &remote.ViewCalendarParams{
			RemoteUserID: u.Remote.ID,
			StartTime:    start,
			EndTime:      end,
		})
	}

	return m.client.DoBatchViewCalendarRequests(params)
}

func (m *mscalendar) notifyUpcomingEvents(mattermostUserID string, events []*remote.Event) {
	var timezone string
	for _, event := range events {
		if event.IsCancelled {
			continue
		}
		upcomingTime := time.Now().Add(upcomingEventNotificationTime)
		start := event.Start.Time()
		diff := start.Sub(upcomingTime)

		if (diff < upcomingEventNotificationWindow) && (diff > -upcomingEventNotificationWindow) {
			var err error
			if timezone == "" {
				timezone, err = m.GetTimezoneByID(mattermostUserID)
				if err != nil {
					m.Logger.Warnf("notifyUpcomingEvents error getting timezone. err=%v", err)
					return
				}
			}

			_, attachment, err := views.RenderUpcomingEventAsAttachment(event, timezone)
			if err != nil {
				m.Logger.Warnf("notifyUpcomingEvent error rendering schedule item. err=%v", err)
				continue
			}

			_, err = m.Poster.DMWithAttachments(mattermostUserID, attachment)
			if err != nil {
				m.Logger.Warnf("notifyUpcomingEvents error creating DM. err=%v", err)
				continue
			}

			// Process channel reminders
			eventMetadata, errMetadata := m.Store.LoadEventMetadata(event.ICalUID)
			if errMetadata != nil && !errors.Is(errMetadata, store.ErrNotFound) {
				m.Logger.With(bot.LogContext{
					"eventID": event.ID,
					"err":     errMetadata.Error(),
				}).Warnf("notifyUpcomingEvents error checking store for channel notifications")
				continue
			}

			if eventMetadata != nil {
				for channelID := range eventMetadata.LinkedChannelIDs {
					post := &model.Post{
						ChannelId: channelID,
						Message:   "Upcoming event",
					}
					attachment, errRender := views.RenderEventAsAttachment(event, timezone, views.ShowTimezoneOption(timezone))
					if errRender != nil {
						m.Logger.With(bot.LogContext{"err": errRender}).Errorf("notifyUpcomingEvents error rendering channel post")
						continue
					}
					model.ParseSlackAttachment(post, []*model.SlackAttachment{attachment})
					errPoster := m.Poster.CreatePost(post)
					if errPoster != nil {
						m.Logger.With(bot.LogContext{"err": errPoster}).Warnf("notifyUpcomingEvents error creating post in channel")
						continue
					}
				}
			}
		}
	}
}

func filterBusyAndAttendeeEvents(events []*remote.Event) []*remote.Event {
	result := []*remote.Event{}
	for _, e := range events {
		// Not setting custom status for events without attendees since those are unlikely to be meetings.
		if e.ShowAs == "busy" && !e.IsCancelled && len(e.Attendees) >= 1 {
			result = append(result, e)
		}
	}
	return result
}

// getMergedEvents accepts a sorted array of events, and returns events after merging them, if overlapping or if the meeting duration is less than StatusSyncJobInterval.
func getMergedEvents(events []*remote.Event) []*remote.Event {
	if len(events) <= 1 {
		return events
	}

	idx := 0
	for i := 1; i < len(events); i++ {
		if areEventsMergeable(events[idx], events[i]) {
			events[idx].End = events[i].End
		} else {
			idx++
			events[idx] = events[i]
		}
	}

	return events[0 : idx+1]
}

/*
areEventsMergeable function checks if two events can be merged into a single event.
There are two conditions that are being checked in the function:
  - If two events overlap, the end time of event1 will be
    greater than or equal to event2 and we can merge those events into a single event.
    For e.g.- event1: 1:01–1:04, event2: 1:03–1:05. Final event: 1:01–1:05.
  - If the difference between event1 end time and event1 start time isor equal to StatusSyncJobInterval and the difference between event2 start time and event1 end time is less than or equal to StatusSyncJobInterval. This is done to merge those events that occur within the time span of StatusSyncJobInterval.
    For e.g.- event1: 1:01–1:02, event2: 1:03–1:05, StatusSyncJobInterval: 5 mins. Final event: 1:01–1:05.
    This is done to avoid skipping of event2 as both events are fetched together in a single API call when the job runs every 5 minutes.
*/
func areEventsMergeable(event1, event2 *remote.Event) bool {
	return (event1.End.Time().UnixMicro() >= event2.Start.Time().UnixMicro()) || (event1.End.Time().Sub(event1.Start.Time()) <= StatusSyncJobInterval && event2.Start.Time().Sub(event1.End.Time()) <= StatusSyncJobInterval)
}
