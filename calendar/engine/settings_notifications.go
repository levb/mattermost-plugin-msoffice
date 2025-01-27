// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/settingspanel"
)

type notificationSetting struct {
	getCal      func(string) Engine
	title       string
	description string
	id          string
	dependsOn   string
}

func NewNotificationsSetting(getCal func(string) Engine) settingspanel.Setting {
	return &notificationSetting{
		title:       "Receive notifications of new events",
		description: "Do you want to subscribe to new events and receive a message when they are created?",
		id:          "new_or_updated_event_setting",
		dependsOn:   "",
		getCal:      getCal,
	}
}

func (s *notificationSetting) Set(userID string, value interface{}) error {
	boolValue := false
	if value == "true" {
		boolValue = true
	}

	cal := s.getCal(userID)

	if boolValue {
		_, err := cal.LoadMyEventSubscription()
		if err != nil {
			_, err := cal.CreateMyEventSubscription()
			if err != nil {
				return err
			}
		}

		return nil
	}

	_, err := cal.LoadMyEventSubscription()
	if err == nil {
		return cal.DeleteMyEventSubscription()
	}
	return nil
}

func (s *notificationSetting) Get(userID string) (interface{}, error) {
	cal := s.getCal(userID)
	_, err := cal.LoadMyEventSubscription()
	if err == nil {
		return "true", nil
	}

	return "false", nil
}

func (s *notificationSetting) GetID() string {
	return s.id
}

func (s *notificationSetting) GetTitle() string {
	return s.title
}

func (s *notificationSetting) GetDescription() string {
	return s.description
}

func (s *notificationSetting) GetDependency() string {
	return s.dependsOn
}

func (s *notificationSetting) getActionStyle(actionValue, currentValue string) string {
	if actionValue == currentValue {
		return "primary"
	}
	return "default"
}

func (s *notificationSetting) GetSlackAttachments(userID, settingHandler string, disabled bool) (*model.SlackAttachment, error) {
	title := fmt.Sprintf("Setting: %s", s.title)
	currentValueMessage := "Disabled"

	actions := []*model.PostAction{}
	if !disabled {
		currentValue, err := s.Get(userID)
		if err != nil {
			return nil, err
		}

		currentTextValue := "No"
		if currentValue == "true" {
			currentTextValue = "Yes"
		}
		currentValueMessage = fmt.Sprintf("**Current value:** %s", currentTextValue)

		actionTrue := model.PostAction{
			Name:  "Yes",
			Style: s.getActionStyle("true", currentValue.(string)),
			Integration: &model.PostActionIntegration{
				URL: settingHandler,
				Context: map[string]interface{}{
					settingspanel.ContextIDKey:          s.id,
					settingspanel.ContextButtonValueKey: "true",
				},
			},
		}

		actionFalse := model.PostAction{
			Name:  "No",
			Style: s.getActionStyle("false", currentValue.(string)),
			Integration: &model.PostActionIntegration{
				URL: settingHandler,
				Context: map[string]interface{}{
					settingspanel.ContextIDKey:          s.id,
					settingspanel.ContextButtonValueKey: "false",
				},
			},
		}
		actions = []*model.PostAction{&actionTrue, &actionFalse}
	}

	text := fmt.Sprintf("%s\n%s", s.description, currentValueMessage)
	sa := model.SlackAttachment{
		Title:    title,
		Text:     text,
		Actions:  actions,
		Fallback: fmt.Sprintf("%s: %s", title, text),
	}

	return &sa, nil
}

func (s *notificationSetting) IsDisabled(foreignValue interface{}) bool {
	return foreignValue == "false"
}
