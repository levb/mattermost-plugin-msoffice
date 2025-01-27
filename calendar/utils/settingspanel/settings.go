// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package settingspanel

import (
	"errors"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/kvstore"
)

type Setting interface {
	Set(userID string, value interface{}) error
	Get(userID string) (interface{}, error)
	GetID() string
	GetDependency() string
	IsDisabled(foreignValue interface{}) bool
	GetTitle() string
	GetDescription() string
	GetSlackAttachments(userID, settingHandler string, disabled bool) (*model.SlackAttachment, error)
}

type Panel interface {
	Set(userID, settingID string, value interface{}) error
	Print(userID string)
	ToPost(userID string) (*model.Post, error)
	Clear(userID string) error
	URL() string
	GetSettingIDs() []string
}

type SettingStore interface {
	SetSetting(userID, settingID string, value interface{}) error
	GetSetting(userID, settingID string) (interface{}, error)
}

type PanelStore interface {
	SetPanelPostID(userID string, postIDs string) error
	GetPanelPostID(userID string) (string, error)
	DeletePanelPostID(userID string) error
}

type panel struct {
	poster         bot.Poster
	logger         bot.Logger
	store          PanelStore
	settings       map[string]Setting
	settingHandler string
	pluginURL      string
	settingKeys    []string
}

func NewSettingsPanel(settings []Setting, poster bot.Poster, logger bot.Logger, store PanelStore, settingHandler, pluginURL string) Panel {
	settingsMap := make(map[string]Setting)
	settingKeys := []string{}
	for _, s := range settings {
		settingsMap[s.GetID()] = s
		settingKeys = append(settingKeys, s.GetID())
	}

	return &panel{
		settings:       settingsMap,
		settingKeys:    settingKeys,
		poster:         poster,
		logger:         logger,
		store:          store,
		settingHandler: settingHandler,
		pluginURL:      pluginURL,
	}
}

func (p *panel) Set(userID, settingID string, value interface{}) error {
	s, ok := p.settings[settingID]
	if !ok {
		return errors.New("cannot find setting " + settingID)
	}

	err := s.Set(userID, value)
	if err != nil {
		return err
	}
	return nil
}

func (p *panel) GetSettingIDs() []string {
	return p.settingKeys
}

func (p *panel) URL() string {
	return p.settingHandler
}

func (p *panel) Print(userID string) {
	err := p.cleanPreviousSettingsPosts(userID)
	if err != nil {
		p.logger.Warnf("could not clean previous setting post. err=%v", err)
	}

	sas := []*model.SlackAttachment{}
	for _, key := range p.settingKeys {
		s := p.settings[key]
		sa, loopErr := s.GetSlackAttachments(userID, p.pluginURL+p.settingHandler, p.isSettingDisabled(userID, s))
		if loopErr != nil {
			p.logger.Warnf("error creating the slack attachment. err=%v", loopErr)
			continue
		}
		sas = append(sas, sa)
	}
	postID, err := p.poster.DMWithAttachments(userID, sas...)
	if err != nil {
		p.logger.Warnf("error creating the message. err=%v", err)
		return
	}

	err = p.store.SetPanelPostID(userID, postID)
	if err != nil {
		p.logger.Warnf("could not set the post IDs. err=%v", err)
	}
}

func (p *panel) ToPost(userID string) (*model.Post, error) {
	post := &model.Post{}

	sas := []*model.SlackAttachment{}
	for _, key := range p.settingKeys {
		s := p.settings[key]
		sa, err := s.GetSlackAttachments(userID, p.pluginURL+p.settingHandler, p.isSettingDisabled(userID, s))
		if err != nil {
			p.logger.Warnf("error creating the slack attachment for setting %s. err=%v", s.GetID(), err)
			continue
		}
		sas = append(sas, sa)
	}

	model.ParseSlackAttachment(post, sas)
	return post, nil
}

func (p *panel) cleanPreviousSettingsPosts(userID string) error {
	postID, err := p.store.GetPanelPostID(userID)
	if err == kvstore.ErrNotFound {
		return nil
	}

	if err != nil {
		return err
	}

	err = p.poster.DeletePost(postID)
	if err != nil {
		p.logger.Warnf("could not delete setting post. err=%v", err)
	}

	err = p.store.DeletePanelPostID(userID)
	if err != nil {
		return err
	}

	return nil
}

func (p *panel) Clear(userID string) error {
	return p.cleanPreviousSettingsPosts(userID)
}

func (p *panel) isSettingDisabled(userID string, s Setting) bool {
	dependencyID := s.GetDependency()
	if dependencyID == "" {
		return false
	}
	dependency, ok := p.settings[dependencyID]
	if !ok {
		p.logger.Warnf("settings dependency %s not found", dependencyID)
		return false
	}

	value, err := dependency.Get(userID)
	if err != nil {
		p.logger.Warnf("cannot get dependency %s value. err=%v", dependencyID, err)
		return false
	}
	return s.IsDisabled(value)
}
