// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package msgraph

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"

	msgraph "github.com/yaegashi/msgraph.go/v1.0"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
)

const Kind = "mscalendar"

type impl struct {
	conf   *config.Config
	logger bot.Logger
}

func init() {
	remote.Makers[Kind] = NewRemote
}

func NewRemote(conf *config.Config, logger bot.Logger) remote.Remote {
	return &impl{
		conf:   conf,
		logger: logger,
	}
}

// MakeClient creates a new client for user-delegated permissions.
func (r *impl) makeClient(ctx context.Context, token *oauth2.Token, mattermostUserID string, poster bot.Poster, userTokenHelpers remote.UserTokenHelpers) remote.Client {
	httpClient := r.NewOAuth2Config().Client(ctx, token)
	c := &client{
		conf:             r.conf,
		ctx:              ctx,
		httpClient:       httpClient,
		Logger:           r.logger,
		rbuilder:         msgraph.NewClient(httpClient),
		tokenHelpers:     userTokenHelpers,
		mattermostUserID: mattermostUserID,
		Poster:           poster,
	}

	return c
}

// MakeUserClient creates a new client having user-delegated permissions with refreshed token.
func (r *impl) MakeUserClient(ctx context.Context, oauthToken *oauth2.Token, mattermostUserID string, poster bot.Poster, userTokenHelpers remote.UserTokenHelpers) remote.Client {
	config := r.NewOAuth2Config()

	token, err := userTokenHelpers.RefreshAndStoreToken(oauthToken, config, mattermostUserID)
	if err != nil {
		r.logger.Warnf("Not able to refresh or store the token", "error", err.Error())
		return &client{}
	}

	return r.makeClient(ctx, token, mattermostUserID, poster, userTokenHelpers)
}

// MakeSuperuserClient creates a new client used for app-only permissions.
func (r *impl) MakeSuperuserClient(ctx context.Context) (remote.Client, error) {
	httpClient := &http.Client{
		Timeout: time.Second * 60,
	}
	c := &client{
		conf:       r.conf,
		ctx:        ctx,
		httpClient: httpClient,
		Logger:     r.logger,
		rbuilder:   msgraph.NewClient(httpClient),
	}
	token, err := c.GetSuperuserToken()
	if err != nil {
		return nil, err
	}

	o := &oauth2.Token{
		AccessToken: token,
		TokenType:   "Bearer",
	}
	return r.makeClient(ctx, o, "", nil, nil), nil
}

func (r *impl) NewOAuth2Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     r.conf.OAuth2ClientID,
		ClientSecret: r.conf.OAuth2ClientSecret,
		RedirectURL:  r.conf.PluginURL + config.FullPathOAuth2Redirect,
		Scopes: []string{
			"offline_access",
			"User.Read",
			"Calendars.ReadWrite",
			"Calendars.ReadWrite.Shared",
			"MailboxSettings.Read",
		},
		Endpoint: microsoft.AzureADEndpoint(r.conf.OAuth2Authority),
	}
}

func (r *impl) CheckConfiguration(cfg config.StoredConfig) error {
	if cfg.OAuth2ClientID == "" || cfg.OAuth2ClientSecret == "" || cfg.OAuth2Authority == "" {
		return fmt.Errorf("OAuth2 credentials to be set in the config")
	}

	return nil
}
