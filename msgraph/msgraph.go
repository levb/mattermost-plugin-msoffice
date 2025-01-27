// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package msgraph

import (
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
)

const (
	ProviderMSCalendar            = Kind
	ProviderMSCalendarDisplayName = "Microsoft Calendar"
	ProviderMSCalendarRepository  = "mattermost-plugin-mscalendar"
)

func GetMSCalendarProviderConfig() config.ProviderConfig {
	return config.ProviderConfig{
		Name:        ProviderMSCalendar,
		DisplayName: ProviderMSCalendarDisplayName,
		Repository:  ProviderMSCalendarRepository,

		CommandTrigger: ProviderMSCalendar,

		TelemetryShortName: ProviderMSCalendar,

		BotUsername:    ProviderMSCalendar,
		BotDisplayName: ProviderMSCalendarDisplayName,

		Features: config.ProviderFeatures{
			EncryptedStore:     false,
			EventNotifications: true,
		},
	}
}
