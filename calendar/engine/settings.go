// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/settingspanel"
)

type Settings interface {
	PrintSettings(userID string)
	ClearSettingsPosts(userID string)
}

func (m *mscalendar) PrintSettings(userID string) {
	m.SettingsPanel.Print(userID)
}

func (m *mscalendar) ClearSettingsPosts(userID string) {
	err := m.SettingsPanel.Clear(userID)
	if err != nil {
		m.Logger.Warnf("Error clearing settings posts. err=%v", err)
	}
}

func NewSettingsPanel(bot bot.Bot, panelStore settingspanel.PanelStore, settingStore settingspanel.SettingStore, settingsHandler, pluginURL string, getCal func(userID string) Engine, providerFeatures config.ProviderFeatures) settingspanel.Panel {
	settings := []settingspanel.Setting{}
	settings = append(settings, settingspanel.NewOptionSetting(
		store.UpdateStatusFromOptionsSettingID,
		"Update Status",
		"Do you want to update your status on Mattermost when you are in a meeting?",
		"",
		store.NotSetStatusOption,
		[]string{store.AwayStatusOption, store.DNDStatusOption, store.NotSetStatusOption},
		settingStore,
	))
	settings = append(settings, settingspanel.NewBoolSetting(
		store.GetConfirmationSettingID,
		"Get Confirmation",
		"Do you want to get a confirmation before automatically updating your status?",
		store.UpdateStatusFromOptionsSettingID,
		settingStore,
	))
	settings = append(settings, settingspanel.NewBoolSetting(
		store.SetCustomStatusSettingID,
		"Set Custom Status",
		"Do you want to set custom status automatically on Mattermost when you are in a meeting?",
		"",
		settingStore,
	))
	settings = append(settings, settingspanel.NewBoolSetting(
		store.ReceiveRemindersSettingID,
		"Receive Reminders",
		"Do you want to receive reminders for upcoming events?",
		"",
		settingStore,
	))
	if providerFeatures.EventNotifications {
		settings = append(settings, NewNotificationsSetting(getCal))
	}
	settings = append(settings, NewDailySummarySetting(
		settingStore,
		func(userID string) (string, error) { return getCal(userID).GetTimezone(NewUser(userID)) },
	))
	return settingspanel.NewSettingsPanel(settings, bot, bot, panelStore, settingsHandler, pluginURL)
}
