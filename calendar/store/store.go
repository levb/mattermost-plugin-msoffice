// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package store

import (
	"time"

	"github.com/mattermost/mattermost/server/public/plugin"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/tracker"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/flow"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/settingspanel"
)

const (
	UserKeyPrefix             = "user_"
	UserIndexKeyPrefix        = "userindex_"
	MattermostUserIDKeyPrefix = "mmuid_"
	OAuth2KeyPrefix           = "oauth2_"
	SubscriptionKeyPrefix     = "sub_"
	EventKeyPrefix            = "ev_"
	WelcomeKeyPrefix          = "welcome_"
	SettingsPanelPrefix       = "settings_panel_"
)

const OAuth2KeyExpiration = 15 * time.Minute

var ErrNotFound = kvstore.ErrNotFound

type Store interface {
	UserStore
	OAuth2StateStore
	SubscriptionStore
	EventStore
	WelcomeStore
	flow.Store
	settingspanel.SettingStore
	settingspanel.PanelStore
}

type pluginStore struct {
	basicKV            kvstore.KVStore
	oauth2KV           kvstore.KVStore
	userKV             kvstore.KVStore
	mattermostUserIDKV kvstore.KVStore
	userIndexKV        kvstore.KVStore
	subscriptionKV     kvstore.KVStore
	eventKV            kvstore.KVStore
	welcomeIndexKV     kvstore.KVStore
	settingsPanelKV    kvstore.KVStore
	Logger             bot.Logger
	Poster             bot.Poster
	Tracker            tracker.Tracker
}

func NewPluginStore(api plugin.API, logger bot.Logger, poster bot.Poster, tracker tracker.Tracker, enableEncryption bool, encryptionKey []byte) Store {
	basicKV := kvstore.NewPluginStore(api)
	oauth2KV := kvstore.NewHashedKeyStore(kvstore.NewOneTimePluginStore(api, OAuth2KeyExpiration), OAuth2KeyPrefix)
	user2KV := kvstore.NewHashedKeyStore(basicKV, UserKeyPrefix)

	if enableEncryption {
		oauth2KV = kvstore.NewEncryptedKeyStore(oauth2KV, encryptionKey)
		user2KV = kvstore.NewEncryptedKeyStore(user2KV, encryptionKey)
	}

	return &pluginStore{
		basicKV:            basicKV,
		userKV:             user2KV,
		userIndexKV:        kvstore.NewHashedKeyStore(basicKV, UserIndexKeyPrefix),
		mattermostUserIDKV: kvstore.NewHashedKeyStore(basicKV, MattermostUserIDKeyPrefix),
		subscriptionKV:     kvstore.NewHashedKeyStore(basicKV, SubscriptionKeyPrefix),
		eventKV:            kvstore.NewHashedKeyStore(basicKV, EventKeyPrefix),
		oauth2KV:           oauth2KV,
		welcomeIndexKV:     kvstore.NewHashedKeyStore(basicKV, WelcomeKeyPrefix),
		settingsPanelKV:    kvstore.NewHashedKeyStore(basicKV, SettingsPanelPrefix),
		Logger:             logger,
		Poster:             poster,
		Tracker:            tracker,
	}
}
