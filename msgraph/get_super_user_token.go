// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package msgraph

import (
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

type AuthResponse struct {
	TokenType   string `json:"token_type"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func (c *client) GetSuperuserToken() (string, error) {
	u := "https://login.microsoftonline.com/" + url.PathEscape(c.conf.OAuth2Authority) + "/oauth2/v2.0/token"
	res := AuthResponse{}

	data := url.Values{}
	data.Set("client_id", c.conf.OAuth2ClientID)
	data.Set("scope", "https://graph.microsoft.com/.default")
	data.Set("client_secret", c.conf.OAuth2ClientSecret)
	data.Set("grant_type", "client_credentials")

	_, err := c.CallFormPost(http.MethodPost, u, data, &res)
	if err != nil {
		return "", errors.Wrap(err, "msgraph GetSuperuserToken")
	}

	return res.AccessToken, nil
}
