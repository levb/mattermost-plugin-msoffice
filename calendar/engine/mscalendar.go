// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/tracker"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/settingspanel"
)

type Engine interface {
	Availability
	Calendar
	EventResponder
	Subscriptions
	Users
	Welcomer
	Settings
	DailySummary
}

// Dependencies contains all API dependencies
type Dependencies struct {
	Logger            bot.Logger
	PluginAPI         PluginAPI
	Poster            bot.Poster
	Remote            remote.Remote
	Store             store.Store
	SettingsPanel     settingspanel.Panel
	IsAuthorizedAdmin func(string) (bool, error)
	Welcomer          Welcomer
	Tracker           tracker.Tracker
}

type PluginAPI interface {
	GetMattermostUser(mattermostUserID string) (*model.User, error)
	GetMattermostUserByUsername(mattermostUsername string) (*model.User, error)
	GetMattermostUserStatus(mattermostUserID string) (*model.Status, error)
	GetMattermostUserStatusesByIds(mattermostUserIDs []string) ([]*model.Status, error)
	IsSysAdmin(mattermostUserID string) (bool, error)
	UpdateMattermostUserStatus(mattermostUserID, status string) (*model.Status, error)
	UpdateMattermostUserCustomStatus(mattermostUserID string, customStatus *model.CustomStatus) *model.AppError
	RemoveMattermostUserCustomStatus(mattermostUserID string) *model.AppError
	GetPost(postID string) (*model.Post, error)
	CanLinkEventToChannel(channelID, userID string) bool
	SearchLinkableChannelForUser(teamID, mattermostUserID, search string) ([]*model.Channel, error)
	GetMattermostUserTeams(mattermostUserID string) ([]*model.Team, error)
	PublishWebsocketEvent(mattermostUserID, event string, payload map[string]any)
}

type Env struct {
	*config.Config
	*Dependencies
}

type mscalendar struct {
	Env

	actingUser *User
	client     remote.Client
}

// copy returns a copy of the calendar engine
func (m mscalendar) copy() *mscalendar {
	user := *m.actingUser
	client := m.client
	return &mscalendar{
		Env:        m.Env,
		actingUser: &user,
		client:     client,
	}
}

func New(env Env, actingMattermostUserID string) Engine {
	return &mscalendar{
		Env:        env,
		actingUser: NewUser(actingMattermostUserID),
	}
}
