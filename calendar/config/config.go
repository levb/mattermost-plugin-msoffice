// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package config

import "github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"

var Provider ProviderConfig

// StoredConfig represents the data stored in and managed with the Mattermost
// config.
type StoredConfig struct {
	OAuth2Authority    string
	OAuth2ClientID     string
	OAuth2ClientSecret string
	bot.Config
	EnableStatusSync   bool
	EnableDailySummary bool

	EncryptionKey string
}

type ProviderFeatures struct {
	EncryptedStore     bool
	EventNotifications bool
}

// ProviderConfig represents the specific configuration that changes when building for different
// calendar providers.
type ProviderConfig struct {
	Name               string
	DisplayName        string
	Repository         string
	CommandTrigger     string
	TelemetryShortName string
	BotUsername        string
	BotDisplayName     string
	Features           ProviderFeatures
}

// Config represents the metadata handed to all request runners (command,
// http).
type Config struct {
	PluginID               string
	BuildDate              string
	BuildHash              string
	BuildHashShort         string
	MattermostSiteHostname string
	MattermostSiteURL      string
	PluginURL              string
	PluginURLPath          string
	PluginVersion          string
	StoredConfig
	Provider ProviderConfig
}

func (c *Config) GetNotificationURL() string {
	return c.PluginURL + FullPathEventNotification
}
