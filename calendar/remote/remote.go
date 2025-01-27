// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package remote

import (
	"context"
	"errors"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
)

var (
	ErrSuperUserClientNotSupported = errors.New("superuser client is not supported")
	ErrNotImplemented              = errors.New("not implemented")
)

type Remote interface {
	MakeUserClient(context.Context, *oauth2.Token, string, bot.Poster, UserTokenHelpers) Client
	MakeSuperuserClient(ctx context.Context) (Client, error)
	NewOAuth2Config() *oauth2.Config
	HandleWebhook(http.ResponseWriter, *http.Request) []*Notification
	CheckConfiguration(configuration config.StoredConfig) error
}

var Makers = map[string]func(*config.Config, bot.Logger) Remote{}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
