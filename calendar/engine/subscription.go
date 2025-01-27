// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
)

type Subscriptions interface {
	CreateMyEventSubscription() (*store.Subscription, error)
	RenewMyEventSubscription() (*store.Subscription, error)
	DeleteOrphanedSubscription(*store.Subscription) error
	DeleteMyEventSubscription() error
	ListRemoteSubscriptions() ([]*remote.Subscription, error)
	LoadMyEventSubscription() (*store.Subscription, error)
}

func (m *mscalendar) CreateMyEventSubscription() (*store.Subscription, error) {
	err := m.Filter(withClient)
	if err != nil {
		return nil, fmt.Errorf("error withClient in CreateMyEventSubscription: %w", err)
	}

	sub, err := m.client.CreateMySubscription(m.Config.GetNotificationURL(), m.actingUser.Remote.ID)
	if err != nil {
		return nil, err
	}

	storedSub := &store.Subscription{
		Remote:              sub,
		MattermostCreatorID: m.actingUser.MattermostUserID,
		PluginVersion:       m.Config.PluginVersion,
	}
	err = m.Store.StoreUserSubscription(m.actingUser.User, storedSub)
	if err != nil {
		return nil, err
	}

	return storedSub, nil
}

func (m *mscalendar) LoadMyEventSubscription() (*store.Subscription, error) {
	err := m.Filter(withActingUserExpanded)
	if err != nil {
		return nil, err
	}

	// TODO: if m.actingUser.Settings.EventSubscriptionID is empty, there's no sub

	storedSub, err := m.Store.LoadSubscription(m.actingUser.Settings.EventSubscriptionID)
	if err != nil {
		return nil, err
	}
	return storedSub, err
}

func (m *mscalendar) ListRemoteSubscriptions() ([]*remote.Subscription, error) {
	err := m.Filter(withClient)
	if err != nil {
		return nil, fmt.Errorf("error withClient in ListRemoteSubscriptions: %w", err)
	}
	subs, err := m.client.ListSubscriptions()
	if err != nil {
		return nil, err
	}
	return subs, nil
}

func (m *mscalendar) RenewMyEventSubscription() (*store.Subscription, error) {
	err := m.Filter(withClient)
	if err != nil {
		return nil, fmt.Errorf("error withClient in RenewMyEventSubscription: %w", err)
	}

	subscriptionID := m.actingUser.Settings.EventSubscriptionID
	if subscriptionID == "" {
		return nil, nil
	}

	sub, err := m.Store.LoadSubscription(subscriptionID)
	if err != nil {
		return nil, errors.Wrap(err, "error loading subscription")
	}

	renewed, err := m.client.RenewSubscription(m.Config.GetNotificationURL(), m.actingUser.Remote.ID, sub.Remote)
	if err != nil {
		if strings.Contains(err.Error(), "The object was not found") {
			err = m.Store.DeleteUserSubscription(m.actingUser.User, subscriptionID)
			if err != nil {
				return nil, err
			}

			m.Logger.Infof("Subscription %s for Mattermost user %s has expired. Creating a new subscription now.", subscriptionID, m.actingUser.MattermostUserID)
			return m.CreateMyEventSubscription()
		}
		return nil, err
	}

	storedSub, err := m.Store.LoadSubscription(m.actingUser.Settings.EventSubscriptionID)
	if err != nil {
		return nil, err
	}
	storedSub.Remote = renewed

	err = m.Store.StoreUserSubscription(m.actingUser.User, storedSub)
	if err != nil {
		return nil, err
	}
	return storedSub, err
}

func (m *mscalendar) DeleteMyEventSubscription() error {
	err := m.Filter(withActingUserExpanded)
	if err != nil {
		return err
	}

	subscriptionID := m.actingUser.Settings.EventSubscriptionID

	sub, err := m.Store.LoadSubscription(subscriptionID)
	if err != nil {
		return errors.Wrap(err, "error loading subscription")
	}

	err = m.DeleteOrphanedSubscription(sub)
	if err != nil {
		return err
	}

	err = m.Store.DeleteUserSubscription(m.actingUser.User, subscriptionID)
	if err != nil {
		return errors.WithMessagef(err, "failed to delete subscription %s", subscriptionID)
	}

	return nil
}

func (m *mscalendar) DeleteOrphanedSubscription(sub *store.Subscription) error {
	err := m.Filter(withClient)
	if err != nil {
		return fmt.Errorf("error withClient in DeleteOrphanedSubscription: %w", err)
	}
	err = m.client.DeleteSubscription(sub.Remote)
	if err != nil {
		return errors.WithMessagef(err, "failed to delete subscription %s", sub.Remote.ID)
	}
	return nil
}
