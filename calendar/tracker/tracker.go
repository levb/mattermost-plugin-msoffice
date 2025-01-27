// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package tracker

import (
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/telemetry"
)

const (
	welcomeFlowCompletionEvent = "welcomeFlowCompletion"
	userAuthenticatedEvent     = "userAuthenticated"
	userDeauthenticatedEvent   = "userDeauthenticated"
	automaticStatusUpdateEvent = "automaticStatusUpdate"
	dailySummarySentEvent      = "dailySummarySent"
)

type Tracker interface {
	TrackWelcomeFlowCompletion(userID string)
	TrackUserAuthenticated(userID string)
	TrackUserDeauthenticated(userID string)
	TrackDailySummarySent(userID string)
	TrackAutomaticStatusUpdate(userID, value, location string)
	ReloadConfig(config *model.Config)
}

func New(t telemetry.Tracker) Tracker {
	return &tracker{
		tracker: t,
	}
}

type tracker struct {
	tracker telemetry.Tracker
}

func (t *tracker) ReloadConfig(config *model.Config) {
	if t.tracker != nil {
		t.tracker.ReloadConfig(telemetry.NewTrackerConfig(config))
	}
}

func (t *tracker) TrackWelcomeFlowCompletion(userID string) {
	t.tracker.TrackUserEvent(welcomeFlowCompletionEvent, userID, map[string]interface{}{})
}

func (t *tracker) TrackUserAuthenticated(userID string) {
	t.tracker.TrackUserEvent(userAuthenticatedEvent, userID, map[string]interface{}{})
}

func (t *tracker) TrackUserDeauthenticated(userID string) {
	t.tracker.TrackUserEvent(userDeauthenticatedEvent, userID, map[string]interface{}{})
}

func (t *tracker) TrackDailySummarySent(userID string) {
	t.tracker.TrackUserEvent(dailySummarySentEvent, userID, map[string]interface{}{})
}

func (t *tracker) TrackAutomaticStatusUpdate(userID, value, location string) {
	properties := map[string]interface{}{
		"value":    value,
		"location": location,
	}
	t.tracker.TrackUserEvent(automaticStatusUpdateEvent, userID, properties)
}
