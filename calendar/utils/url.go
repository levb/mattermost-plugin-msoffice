// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package utils

import "net/url"

func IsURL(u string) bool {
	_, err := url.ParseRequestURI(u)
	return err == nil
}
