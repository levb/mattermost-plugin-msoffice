// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package bot

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

type Poster interface {
	// DM posts a simple Direct Message to the specified user
	DM(mattermostUserID, format string, args ...interface{}) (string, error)

	// DMWithAttachments posts a Direct Message that contains Slack attachments.
	// Often used to include post actions.
	DMWithAttachments(mattermostUserID string, attachments ...*model.SlackAttachment) (string, error)

	// DMWithMessageAndAttachments posts a Direct Message that contains Slack attachments and a message.
	DMWithMessageAndAttachments(mattermostUserID, message string, attachments ...*model.SlackAttachment) (string, error)

	// Ephemeral sends an ephemeral message to a user
	Ephemeral(mattermostUserID, channelID, format string, args ...interface{})

	// DMUpdate updates the postID with the formatted message
	DMUpdate(postID, format string, args ...interface{}) error

	// CreatePost creates a post
	CreatePost(post *model.Post) error

	// DeletePost deletes a single post
	DeletePost(postID string) error

	// UpdatePost substitute one post with another
	UpdatePost(post *model.Post) error
}

// DM posts a simple Direct Message to the specified user
func (bot *bot) DM(mattermostUserID, format string, args ...interface{}) (string, error) {
	postID, err := bot.dm(mattermostUserID, &model.Post{
		Message: fmt.Sprintf(format, args...),
	})
	if err != nil {
		return "", err
	}
	return postID, nil
}

// DMWithAttachments posts a Direct Message that contains Slack attachments.
// Often used to include post actions.
func (bot *bot) DMWithAttachments(mattermostUserID string, attachments ...*model.SlackAttachment) (string, error) {
	post := model.Post{}
	model.ParseSlackAttachment(&post, attachments)
	return bot.dm(mattermostUserID, &post)
}

// DMWithMessageAndAttachments posts a Direct Message that contains Slack attachments and a message.
func (bot *bot) DMWithMessageAndAttachments(mattermostUserID, message string, attachments ...*model.SlackAttachment) (string, error) {
	post := model.Post{Message: message}
	model.ParseSlackAttachment(&post, attachments)
	return bot.dm(mattermostUserID, &post)
}

func (bot *bot) dm(mattermostUserID string, post *model.Post) (string, error) {
	channel, err := bot.pluginAPI.GetDirectChannel(mattermostUserID, bot.mattermostUserID)
	if err != nil {
		bot.pluginAPI.LogInfo("Couldn't get bot's DM channel", "user_id", mattermostUserID)
		return "", err
	}
	post.ChannelId = channel.Id
	post.UserId = bot.mattermostUserID
	sentPost, err := bot.pluginAPI.CreatePost(post)
	if err != nil {
		return "", err
	}
	return sentPost.Id, nil
}

// DM posts a simple Direct Message to the specified user
func (bot *bot) dmAdmins(format string, args ...interface{}) error {
	for _, id := range strings.Split(bot.AdminUserIDs, ",") {
		_, err := bot.dm(id, &model.Post{
			Message: fmt.Sprintf(format, args...),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Ephemeral sends an ephemeral message to a user
func (bot *bot) Ephemeral(userID, channelID, format string, args ...interface{}) {
	post := &model.Post{
		UserId:    bot.mattermostUserID,
		ChannelId: channelID,
		Message:   fmt.Sprintf(format, args...),
	}
	_ = bot.pluginAPI.SendEphemeralPost(userID, post)
}

func (bot *bot) DMUpdate(postID, format string, args ...interface{}) error {
	post, appErr := bot.pluginAPI.GetPost(postID)
	if appErr != nil {
		return appErr
	}

	post.Message = fmt.Sprintf(format, args...)
	_, appErr = bot.pluginAPI.UpdatePost(post)
	if appErr != nil {
		return appErr
	}

	return nil
}

func (bot *bot) CreatePost(post *model.Post) error {
	post.UserId = bot.mattermostUserID
	_, appErr := bot.pluginAPI.CreatePost(post)
	if appErr != nil {
		return appErr
	}

	return nil
}

func (bot *bot) DeletePost(postID string) error {
	appErr := bot.pluginAPI.DeletePost(postID)
	if appErr != nil {
		return appErr
	}
	return nil
}

func (bot *bot) UpdatePost(post *model.Post) error {
	_, appErr := bot.pluginAPI.UpdatePost(post)
	if appErr != nil {
		return appErr
	}
	return nil
}
