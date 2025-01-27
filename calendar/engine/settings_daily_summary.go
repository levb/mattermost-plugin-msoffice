// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/settingspanel"
)

type dailySummarySetting struct {
	store       settingspanel.SettingStore
	getTimezone func(userID string) (string, error)
	title       string
	dependsOn   string
	description string
	id          string
	optionsH    []string
	optionsM    []string
	optionsAPM  []string
}

func NewDailySummarySetting(inStore settingspanel.SettingStore, getTimezone func(userID string) (string, error)) settingspanel.Setting {
	os := &dailySummarySetting{
		title:       "Daily Summary",
		description: "When do you want to receive the daily summary?\n If you update this setting, it will automatically update to your the timezone currently set on your calendar.",
		id:          store.DailySummarySettingID,
		dependsOn:   "",
		store:       inStore,
		getTimezone: getTimezone,
	}
	os.optionsH = []string{"12"}
	for i := 1; i < 12; i++ {
		os.optionsH = append(os.optionsH, fmt.Sprintf("%d", i))
	}

	os.optionsM = []string{}
	for i := 0; i < 4; i++ {
		os.optionsM = append(os.optionsM, fmt.Sprintf("%02d", i*15))
	}

	os.optionsAPM = []string{"AM", "PM"}

	return os
}

func (s *dailySummarySetting) Set(userID string, value interface{}) error {
	_, ok := value.(string)
	if !ok {
		return errors.New("trying to set Daily Summary Setting without a string value")
	}
	err := s.store.SetSetting(userID, s.id, value)
	if err != nil {
		return err
	}

	return nil
}

func (s *dailySummarySetting) Get(userID string) (interface{}, error) {
	value, err := s.store.GetSetting(userID, s.id)
	if err != nil {
		return nil, err
	}

	_, ok := value.(*store.DailySummaryUserSettings)
	if !ok {
		return nil, errors.New("current value is not a Daily Summary Setting")
	}

	return value, nil
}

func (s *dailySummarySetting) GetID() string {
	return s.id
}

func (s *dailySummarySetting) GetTitle() string {
	return s.title
}

func (s *dailySummarySetting) GetDescription() string {
	return s.description
}

func (s *dailySummarySetting) GetDependency() string {
	return s.dependsOn
}

func (s *dailySummarySetting) GetSlackAttachments(userID, settingHandler string, disabled bool) (*model.SlackAttachment, error) {
	title := fmt.Sprintf("Setting: %s", s.title)
	currentValueMessage := "Disabled"

	actions := []*model.PostAction{}

	if disabled {
		text := fmt.Sprintf("%s\n%s", s.description, currentValueMessage)
		sa := model.SlackAttachment{
			Title:    title,
			Text:     text,
			Actions:  actions,
			Fallback: fmt.Sprintf("%s: %s", title, text),
		}
		return &sa, nil
	}

	dsumRaw, err := s.Get(userID)
	if err != nil {
		return nil, err
	}
	dsum := dsumRaw.(*store.DailySummaryUserSettings)

	currentH := "8"
	currentM := "00"
	currentAPM := "AM"
	fullTime := "8:00AM"
	currentEnable := false

	if dsum != nil {
		fullTime = dsum.PostTime
		currentEnable = dsum.Enable
		splitted := strings.Split(fullTime, ":")
		currentH = splitted[0]
		currentM = splitted[1][:2]
		currentAPM = splitted[1][2:]
	}

	timezone, err := s.getTimezone(userID)
	if err != nil {
		return nil, fmt.Errorf("could not load the timezone. err=%v", err)
	}
	fullTime = fullTime + " " + timezone

	actionOptionsH := model.PostAction{
		Name: "H:",
		Integration: &model.PostActionIntegration{
			URL: settingHandler,
			Context: map[string]interface{}{
				settingspanel.ContextIDKey: s.id,
			},
		},
		Type:          "select",
		Options:       s.makeHOptions(currentM, currentAPM, timezone),
		DefaultOption: fullTime,
	}

	actionOptionsM := model.PostAction{
		Name: "M:",
		Integration: &model.PostActionIntegration{
			URL: settingHandler,
			Context: map[string]interface{}{
				settingspanel.ContextIDKey: s.id,
			},
		},
		Type:          "select",
		Options:       s.makeMOptions(currentH, currentAPM, timezone),
		DefaultOption: fullTime,
	}

	actionOptionsAPM := model.PostAction{
		Name: "AM/PM:",
		Integration: &model.PostActionIntegration{
			URL: settingHandler,
			Context: map[string]interface{}{
				settingspanel.ContextIDKey: s.id,
			},
		},
		Type:          "select",
		Options:       s.makeAPMOptions(currentH, currentM, timezone),
		DefaultOption: fullTime,
	}

	if currentEnable {
		actions = []*model.PostAction{&actionOptionsH, &actionOptionsM, &actionOptionsAPM}
	}

	buttonText := "Enable"
	enable := "true"
	if currentEnable {
		buttonText = "Disable"
		enable = "false"
	}
	actionToggle := model.PostAction{
		Name: buttonText,
		Integration: &model.PostActionIntegration{
			URL: settingHandler,
			Context: map[string]interface{}{
				settingspanel.ContextIDKey:          s.id,
				settingspanel.ContextButtonValueKey: enable + " " + timezone,
			},
		},
	}

	actions = append(actions, &actionToggle)

	sa := model.SlackAttachment{
		Title:    title,
		Text:     s.description,
		Actions:  actions,
		Fallback: fmt.Sprintf("%s: %s", title, s.description),
	}
	return &sa, nil
}

func (s *dailySummarySetting) IsDisabled(foreignValue interface{}) bool {
	return foreignValue == "false"
}

func (s *dailySummarySetting) makeHOptions(minute, apm, timezone string) []*model.PostActionOptions {
	out := []*model.PostActionOptions{}
	for _, o := range s.optionsH {
		out = append(out, &model.PostActionOptions{
			Text:  o,
			Value: fmt.Sprintf("%s:%s%s %s", o, minute, apm, timezone),
		})
	}
	return out
}

func (s *dailySummarySetting) makeMOptions(hour, apm, timezone string) []*model.PostActionOptions {
	out := []*model.PostActionOptions{}
	for _, o := range s.optionsM {
		out = append(out, &model.PostActionOptions{
			Text:  o,
			Value: fmt.Sprintf("%s:%s%s %s", hour, o, apm, timezone),
		})
	}
	return out
}

func (s *dailySummarySetting) makeAPMOptions(hour, minute, timezone string) []*model.PostActionOptions {
	out := []*model.PostActionOptions{}

	for _, o := range s.optionsAPM {
		out = append(out, &model.PostActionOptions{
			Text:  o,
			Value: fmt.Sprintf("%s:%s%s %s", hour, minute, o, timezone),
		})
	}

	return out
}
