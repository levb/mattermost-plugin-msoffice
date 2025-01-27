// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import "github.com/pkg/errors"

type filterf func(*mscalendar) error

func (m *mscalendar) Filter(filters ...filterf) error {
	for _, filter := range filters {
		err := filter(m)
		if err != nil {
			return err
		}
	}
	return nil
}

// FilterCopy creates a copy of the calendar engine and applies filters to it
func (m *mscalendar) FilterCopy(filters ...filterf) (*mscalendar, error) {
	engine := m.copy()
	if err := engine.Filter(filters...); err != nil {
		return nil, errors.Wrap(err, "error filtering engine copy")
	}

	return engine, nil
}

func withActingUserExpanded(m *mscalendar) error {
	return m.ExpandUser(m.actingUser)
}

func withUserExpanded(user *User) func(m *mscalendar) error {
	return func(m *mscalendar) error {
		return m.ExpandUser(user)
	}
}

func withRemoteUser(user *User) func(m *mscalendar) error {
	return func(m *mscalendar) error {
		return m.ExpandRemoteUser(user)
	}
}

func withActingUser(mattermostUserID string) func(m *mscalendar) error {
	return func(m *mscalendar) error {
		m.actingUser = NewUser(mattermostUserID)
		m.client = nil
		return nil
	}
}

func withClient(m *mscalendar) error {
	if m.client != nil {
		return nil
	}

	client, err := m.MakeClient()
	if err != nil {
		return err
	}

	m.client = client
	return nil
}

func withSuperuserClient(m *mscalendar) error {
	if m.client != nil {
		return nil
	}

	client, err := m.MakeSuperuserClient()
	if err != nil {
		return err
	}

	m.client = client
	return nil
}
