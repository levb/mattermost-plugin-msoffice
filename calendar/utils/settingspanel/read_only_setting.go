// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package settingspanel

import (
	"errors"
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
)

type readOnlySetting struct {
	store       SettingStore
	title       string
	description string
	id          string
	dependsOn   string
}

func NewReadOnlySetting(id string, title string, description string, dependsOn string, store SettingStore) Setting {
	return &readOnlySetting{
		title:       title,
		description: description,
		id:          id,
		dependsOn:   dependsOn,
		store:       store,
	}
}

func (s *readOnlySetting) Set(_ string, _ interface{}) error {
	return nil
}

func (s *readOnlySetting) Get(userID string) (interface{}, error) {
	value, err := s.store.GetSetting(userID, s.id)
	if err != nil {
		return "", err
	}
	stringValue, ok := value.(string)
	if !ok {
		return "", errors.New("current value is not a string")
	}

	return stringValue, nil
}

func (s *readOnlySetting) GetID() string {
	return s.id
}

func (s *readOnlySetting) GetTitle() string {
	return s.title
}

func (s *readOnlySetting) GetDescription() string {
	return s.description
}

func (s *readOnlySetting) GetDependency() string {
	return s.dependsOn
}

func (s *readOnlySetting) GetSlackAttachments(userID, _ string, disabled bool) (*model.SlackAttachment, error) {
	title := fmt.Sprintf("Setting: %s", s.title)
	currentValueMessage := "Disabled"

	if !disabled {
		currentValue, err := s.Get(userID)
		if err != nil {
			return nil, err
		}
		currentValueMessage = fmt.Sprintf("**Current value:** %s", currentValue)
	}

	text := fmt.Sprintf("%s\n%s", s.description, currentValueMessage)
	sa := model.SlackAttachment{
		Title:    title,
		Text:     text,
		Fallback: fmt.Sprintf("%s: %s", title, text),
	}

	return &sa, nil
}

func (s *readOnlySetting) IsDisabled(foreignValue interface{}) bool {
	return foreignValue == "false"
}
