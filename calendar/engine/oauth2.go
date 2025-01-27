// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/oauth2connect"
)

const BotWelcomeMessage = "Bot user connected to account %s."

const (
	RemoteUserAlreadyConnected         = "%s account `%s` is already mapped to Mattermost account `%s`. Please run `/%s disconnect`, while logged in as the Mattermost account"
	RemoteUserAlreadyConnectedDisabled = "%s account `%s` is already mapped to a Mattermost account, but the account is deactivated. Please enable it and run `/%s disconnect`,  while logged in as the other Mattermost account, and try again"
	RemoteUserAlreadyConnectedNotFound = "%s account `%s` is already mapped to a Mattermost account, but the Mattermost user could not be found"
)

type oauth2App struct {
	Env
}

func NewOAuth2App(env Env) oauth2connect.App {
	return &oauth2App{
		Env: env,
	}
}

func (app *oauth2App) InitOAuth2(mattermostUserID string) (url string, err error) {
	user, err := app.Store.LoadUser(mattermostUserID)
	if err == nil {
		return "", fmt.Errorf("user is already connected to %s", user.Remote.Mail)
	}

	conf := app.Remote.NewOAuth2Config()
	state := fmt.Sprintf("%v_%v", model.NewId()[0:15], mattermostUserID)
	err = app.Store.StoreOAuth2State(state)
	if err != nil {
		return "", err
	}

	return conf.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

func (app *oauth2App) CompleteOAuth2(authedUserID, code, state string) error {
	if authedUserID == "" || code == "" || state == "" {
		return errors.New("missing user, code or state")
	}

	oconf := app.Remote.NewOAuth2Config()

	err := app.Store.VerifyOAuth2State(state)
	if err != nil {
		return errors.WithMessage(err, "missing stored state")
	}

	mattermostUserID := strings.Split(state, "_")[1]
	if mattermostUserID != authedUserID {
		return errors.New("not authorized, user ID mismatch")
	}

	ctx := context.Background()
	tok, err := oconf.Exchange(ctx, code)
	if err != nil {
		return err
	}

	client := app.Remote.MakeUserClient(ctx, tok, mattermostUserID, app.Poster, app.Store)
	me, err := client.GetMe()
	if err != nil {
		return err
	}

	uid, err := app.Store.LoadMattermostUserID(me.ID)
	if err == nil {
		user, userErr := app.PluginAPI.GetMattermostUser(uid)
		if userErr == nil {
			msg := fmt.Sprintf(RemoteUserAlreadyConnected, config.Provider.DisplayName, me.Mail, user.Username, config.Provider.CommandTrigger)
			app.Poster.DM(authedUserID, msg)
			return errors.New(msg)
		}

		if userErr == store.ErrNotFound {
			msg := fmt.Sprintf(RemoteUserAlreadyConnectedDisabled, config.Provider.DisplayName, me.Mail, config.Provider.CommandTrigger)
			app.Poster.DM(authedUserID, msg)
			return errors.New(msg)
		}

		// Couldn't fetch connected MM account. Reject connect attempt.
		msg := fmt.Sprintf(RemoteUserAlreadyConnectedNotFound, config.Provider.DisplayName, me.Mail)
		app.Poster.DM(authedUserID, msg)
		return errors.New(msg)
	}

	user, userErr := app.PluginAPI.GetMattermostUser(mattermostUserID)
	if userErr != nil {
		return fmt.Errorf("error retrieving mattermost user (%s): %w", mattermostUserID, userErr)
	}

	u := &store.User{
		PluginVersion:         app.Config.PluginVersion,
		MattermostUserID:      mattermostUserID,
		MattermostUsername:    user.Username,
		MattermostDisplayName: user.GetDisplayName(model.ShowFullName),
		Remote:                me,
		OAuth2Token:           tok,
		Settings:              store.DefaultSettings,
	}

	mailboxSettings, err := client.GetMailboxSettings(me.ID)
	if err != nil {
		return err
	}

	u.Settings.DailySummary = &store.DailySummaryUserSettings{
		PostTime: "8:00AM",
		Timezone: mailboxSettings.TimeZone,
		Enable:   false,
	}

	err = app.Store.StoreUser(u)
	if err != nil {
		return err
	}

	err = app.Store.StoreUserInIndex(u)
	if err != nil {
		return err
	}

	app.Welcomer.AfterSuccessfullyConnect(mattermostUserID, me.Mail)

	return nil
}
