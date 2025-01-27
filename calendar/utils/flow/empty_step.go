// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package flow

import (
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
)

type EmptyStep struct {
	Title   string
	Message string
}

func (s *EmptyStep) PostSlackAttachment(_ string, _ int) *model.SlackAttachment {
	sa := model.SlackAttachment{
		Title:    s.Title,
		Text:     s.Message,
		Fallback: fmt.Sprintf("%s: %s", s.Title, s.Message),
	}

	return &sa
}

func (s *EmptyStep) ResponseSlackAttachment(_ bool) *model.SlackAttachment {
	return nil
}

func (s *EmptyStep) GetPropertyName() string {
	return ""
}

func (s *EmptyStep) ShouldSkip(_ bool) int {
	return 0
}

func (s *EmptyStep) IsEmpty() bool {
	return true
}
