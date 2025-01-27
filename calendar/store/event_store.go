// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package store

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/kvstore"
)

// If event has an end date/time, its record will be set to expire ttlAfterEventEnd
// after its end. Events that have no end-date are created for defaultEventsTTL.
// Expirations are updated when events themselves are updated.
const ttlAfterEventEnd = 30 * 24 * time.Hour // 30 days
const defaultEventTTL = 30 * 24 * time.Hour  // 30 days

type EventMetadata struct {
	LinkedChannelIDs map[string]struct{}
}

type Event struct {
	Remote        *remote.Event
	PluginVersion string
}

type EventStore interface {
	LoadEventMetadata(eventID string) (*EventMetadata, error)
	StoreEventMetadata(eventID string, eventMeta *EventMetadata) error
	DeleteEventMetadata(eventID string) error

	AddLinkedChannelToEvent(eventID, channelID string) error
	DeleteLinkedChannelFromEvent(eventID, channelID string) error

	LoadUserEvent(mattermostUserID, eventID string) (*Event, error)
	StoreUserEvent(mattermostUserID string, event *Event) error
	DeleteUserEvent(mattermostUserID, eventID string) error
}

func eventKey(mattermostUserID, eventID string) string { return mattermostUserID + "_" + eventID }
func eventMetaKey(eventID string) string               { return "metadata_" + eventID }

func (s *pluginStore) LoadUserEvent(mattermostUserID, eventID string) (*Event, error) {
	event := Event{}
	err := kvstore.LoadJSON(s.eventKV, eventKey(mattermostUserID, eventID), &event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (s *pluginStore) AddLinkedChannelToEvent(eventID, channelID string) error {
	eventMeta, err := s.LoadEventMetadata(eventID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}

	if eventMeta == nil {
		eventMeta = &EventMetadata{
			LinkedChannelIDs: make(map[string]struct{}, 1),
		}
	}

	eventMeta.LinkedChannelIDs[channelID] = struct{}{}

	return s.StoreEventMetadata(eventID, eventMeta)
}

func (s *pluginStore) DeleteLinkedChannelFromEvent(eventID, channelID string) error {
	eventMeta, err := s.LoadEventMetadata(eventID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}

	delete(eventMeta.LinkedChannelIDs, channelID)

	return s.StoreEventMetadata(eventID, eventMeta)
}

func (s *pluginStore) StoreEventMetadata(eventID string, eventMeta *EventMetadata) error {
	err := kvstore.StoreJSON(s.eventKV, eventMetaKey(eventID), &eventMeta)
	if err != nil {
		return errors.Wrap(err, "error storing event metadata")
	}
	return nil
}

func (s *pluginStore) LoadEventMetadata(eventID string) (*EventMetadata, error) {
	event := EventMetadata{}
	err := kvstore.LoadJSON(s.eventKV, eventMetaKey(eventID), &event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (s *pluginStore) DeleteEventMetadata(eventID string) error {
	return s.eventKV.Delete(eventMetaKey(eventID))
}

func (s *pluginStore) StoreUserEvent(mattermostUserID string, event *Event) error {
	now := time.Now()
	end := now.Add(defaultEventTTL)
	if event.Remote.End != nil {
		end = event.Remote.End.Time().Add(ttlAfterEventEnd)
		if end.Before(now) {
			// no point storing expired keys
			return nil
		}
	}

	ttl := int64(end.Sub(now).Seconds())
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	err = s.eventKV.StoreTTL(eventKey(mattermostUserID, event.Remote.ICalUID), data, ttl)
	if err != nil {
		return err
	}

	s.Logger.With(bot.LogContext{
		"mattermostUserID": mattermostUserID,
		"eventID":          event.Remote.ID,
		"expires":          end.String(),
	}).Debugf("store: stored user event.")

	return nil
}

func (s *pluginStore) DeleteUserEvent(mattermostUserID, eventID string) error {
	err := s.eventKV.Delete(eventKey(mattermostUserID, eventID))
	if err != nil {
		return err
	}

	s.Logger.With(bot.LogContext{
		"mattermostUserID": mattermostUserID,
		"eventID":          eventID,
	}).Debugf("store: deleted event.")

	return nil
}
