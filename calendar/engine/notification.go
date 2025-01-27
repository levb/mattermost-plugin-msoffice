// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
//  See License for license information.

package engine

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
)

const maxQueueSize = 1024

const (
	FieldSubject        = "Subject"
	FieldBodyPreview    = "BodyPreview"
	FieldImportance     = "Importance"
	FieldDuration       = "Duration"
	FieldWhen           = "When"
	FieldLocation       = "Location"
	FieldAttendees      = "Attendees"
	FieldOrganizer      = "Organizer"
	FieldResponseStatus = "ResponseStatus"
)

const (
	OptionYes          = "Yes"
	OptionNotResponded = "Not responded"
	OptionNo           = "No"
	OptionMaybe        = "Maybe"
)

const (
	ResponseYes   = "accepted"
	ResponseMaybe = "tentativelyAccepted"
	ResponseNo    = "declined"
	ResponseNone  = "notResponded"
)

var importantNotificationChanges = []string{FieldSubject, FieldWhen}

var notificationFieldOrder = []string{
	FieldWhen,
	FieldLocation,
	FieldAttendees,
	FieldImportance,
}

type NotificationProcessor interface {
	Configure(Env)
	Enqueue(notifications ...*remote.Notification) error
	Quit()
}

type notificationProcessor struct {
	Env
	envChan chan Env

	queue chan *remote.Notification
	quit  chan bool
}

func NewNotificationProcessor(env Env) NotificationProcessor {
	processor := &notificationProcessor{
		Env:     env,
		envChan: make(chan (Env)),
		queue:   make(chan (*remote.Notification), maxQueueSize),
		quit:    make(chan (bool)),
	}
	go processor.work()
	return processor
}

func (processor *notificationProcessor) Enqueue(notifications ...*remote.Notification) error {
	for _, n := range notifications {
		select {
		case processor.queue <- n:
		default:
			return fmt.Errorf("webhook notification: queue full, dropped notification")
		}
	}
	return nil
}

func (processor *notificationProcessor) Configure(env Env) {
	processor.envChan <- env
}

func (processor *notificationProcessor) Quit() {
	processor.quit <- true
}

func (processor *notificationProcessor) work() {
	for {
		select {
		case n := <-processor.queue:
			err := processor.processNotification(n)
			if err != nil {
				processor.Logger.With(bot.LogContext{
					"subscriptionID": n.SubscriptionID,
				}).Infof("webhook notification: failed: `%v`.", err)
			}

		case env := <-processor.envChan:
			processor.Env = env

		case <-processor.quit:
			return
		}
	}
}

func (processor *notificationProcessor) processNotification(n *remote.Notification) error {
	sub, err := processor.Store.LoadSubscription(n.SubscriptionID)
	if err != nil {
		return err
	}
	creator, err := processor.Store.LoadUser(sub.MattermostCreatorID)
	if err != nil {
		return err
	}
	if sub.Remote.ID != creator.Settings.EventSubscriptionID {
		return errors.New("subscription is orphaned")
	}
	if sub.Remote.ClientState != "" && sub.Remote.ClientState != n.ClientState {
		return errors.New("unauthorized webhook")
	}

	n.Subscription = sub.Remote
	n.SubscriptionCreator = creator.Remote

	client := processor.Remote.MakeUserClient(context.Background(), creator.OAuth2Token, sub.MattermostCreatorID, processor.Poster, processor.Store)

	if n.RecommendRenew {
		var renewed *remote.Subscription
		renewed, err = client.RenewSubscription(processor.Config.GetNotificationURL(), sub.Remote.CreatorID, n.Subscription)
		if err != nil {
			return err
		}

		storedSub := &store.Subscription{
			Remote:              renewed,
			MattermostCreatorID: creator.MattermostUserID,
			PluginVersion:       processor.Config.PluginVersion,
		}
		err = processor.Store.StoreUserSubscription(creator, storedSub)
		if err != nil {
			return err
		}
		processor.Logger.With(bot.LogContext{
			"MattermostUserID": creator.MattermostUserID,
			"SubscriptionID":   n.SubscriptionID,
		}).Debugf("webhook notification: renewed user subscription.")
	}

	if n.IsBare {
		n, err = client.GetNotificationData(n)
		if err != nil {
			return err
		}
	}

	var sa *model.SlackAttachment
	prior, err := processor.Store.LoadUserEvent(creator.MattermostUserID, n.Event.ICalUID)
	if err != nil && err != store.ErrNotFound {
		return err
	}

	mailSettings, err := client.GetMailboxSettings(sub.Remote.CreatorID)
	if err != nil {
		return err
	}
	timezone := mailSettings.TimeZone

	if prior != nil {
		var changed bool
		changed, sa = processor.updatedEventSlackAttachment(n, prior.Remote, timezone)
		if !changed {
			processor.Logger.With(bot.LogContext{
				"MattermostUserID": creator.MattermostUserID,
				"SubscriptionID":   n.SubscriptionID,
				"ChangeType":       n.ChangeType,
				"EventID":          n.Event.ID,
				"EventICalUID":     n.Event.ICalUID,
			}).Debugf("webhook notification: no changes detected in event.")
			return nil
		}
	} else {
		sa = processor.newEventSlackAttachment(n, timezone)
		prior = &store.Event{}
	}

	_, err = processor.Poster.DMWithAttachments(creator.MattermostUserID, sa)
	if err != nil {
		return err
	}

	prior.Remote = n.Event
	err = processor.Store.StoreUserEvent(creator.MattermostUserID, prior)
	if err != nil {
		return err
	}

	processor.Logger.With(bot.LogContext{
		"MattermostUserID": creator.MattermostUserID,
		"SubscriptionID":   n.SubscriptionID,
	}).Debugf("Notified: %s.", sa.Title)

	return nil
}
