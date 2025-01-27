// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package settingspanel

import "github.com/mattermost/mattermost/server/public/model"

func stringsToOptions(in []string) []*model.PostActionOptions {
	out := []*model.PostActionOptions{}
	for _, o := range in {
		out = append(out, &model.PostActionOptions{
			Text:  o,
			Value: o,
		})
	}
	return out
}
