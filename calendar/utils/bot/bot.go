// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package bot

import (
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/flow"
)

type Bot interface {
	Poster
	Logger
	Admin
	FlowController

	Ensure(stored *model.Bot, iconPath string) error
	WithConfig(Config) Bot
	MattermostUserID() string
	RegisterFlow(flow.Flow, flow.Store)
}

type bot struct {
	pluginAPI        plugin.API
	flow             flow.Flow
	flowStore        flow.Store
	logContext       LogContext
	pluginURL        string
	mattermostUserID string
	displayName      string
	Config
}

func New(api plugin.API, pluginURL string) Bot {
	return &bot{
		pluginAPI: api,
		pluginURL: pluginURL,
	}
}

func (bot *bot) RegisterFlow(flow flow.Flow, flowStore flow.Store) {
	bot.flow = flow
	bot.flowStore = flowStore
}

func (bot *bot) Ensure(stored *model.Bot, iconPath string) error {
	if bot.mattermostUserID != "" {
		// Already done
		return nil
	}
	client := pluginapi.NewClient(bot.pluginAPI, nil) // driver passed as nil, as we don't need it
	botUserID, err := client.Bot.EnsureBot(stored, pluginapi.ProfileImagePath(iconPath))
	if err != nil {
		return errors.Wrap(err, "failed to ensure bot account")
	}
	bot.mattermostUserID = botUserID
	bot.displayName = stored.DisplayName
	return nil
}

func (bot *bot) WithConfig(conf Config) Bot {
	newbot := *bot
	newbot.Config = conf
	return &newbot
}

func (bot *bot) MattermostUserID() string {
	return bot.mattermostUserID
}

func (bot *bot) String() string {
	return bot.displayName
}
