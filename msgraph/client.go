// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package msgraph

import (
	"context"
	"net/http"

	msgraph "github.com/yaegashi/msgraph.go/v1.0"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
)

type client struct {
	// caching the context here since it's a "single-use" client, usually used
	// within a single API request
	ctx context.Context

	httpClient       *http.Client
	rbuilder         *msgraph.GraphServiceRequestBuilder
	mattermostUserID string
	conf             *config.Config
	tokenHelpers     remote.UserTokenHelpers

	bot.Logger
	bot.Poster
}
