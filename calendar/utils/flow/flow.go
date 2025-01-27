// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package flow

import (
	"fmt"
	"strconv"

	"github.com/mattermost/mattermost/server/public/model"
)

type Flow interface {
	Step(i int) Step
	URL() string
	Length() int
	StepDone(userID string, step int, value bool)
	FlowDone(userID string)
}

type Store interface {
	SetProperty(userID, propertyName string, value interface{}) error
	SetPostID(userID, propertyName, postID string) error
	GetPostID(userID, propertyName string) (string, error)
	RemovePostID(userID, propertyName string) error
	GetCurrentStep(userID string) (int, error)
	SetCurrentStep(userID string, step int) error
	DeleteCurrentStep(userID string) error
}

type Step interface {
	PostSlackAttachment(flowHandler string, i int) *model.SlackAttachment
	ResponseSlackAttachment(value bool) *model.SlackAttachment
	GetPropertyName() string
	ShouldSkip(value bool) int
	IsEmpty() bool
}

type SimpleStep struct {
	Title                string
	Message              string
	PropertyName         string
	TrueButtonMessage    string
	FalseButtonMessage   string
	TrueResponseMessage  string
	FalseResponseMessage string
	TrueSkip             int
	FalseSkip            int
}

func (s *SimpleStep) PostSlackAttachment(flowHandler string, i int) *model.SlackAttachment {
	actionTrue := model.PostAction{
		Name: s.TrueButtonMessage,
		Integration: &model.PostActionIntegration{
			URL: flowHandler + "?" + s.PropertyName + "=true&step=" + strconv.Itoa(i),
		},
	}

	actionFalse := model.PostAction{
		Name: s.FalseButtonMessage,
		Integration: &model.PostActionIntegration{
			URL: flowHandler + "?" + s.PropertyName + "=false&step=" + strconv.Itoa(i),
		},
	}

	sa := model.SlackAttachment{
		Title:    s.Title,
		Text:     s.Message,
		Actions:  []*model.PostAction{&actionTrue, &actionFalse},
		Fallback: fmt.Sprintf("%s: %s", s.Title, s.Message),
	}

	return &sa
}

func (s *SimpleStep) ResponseSlackAttachment(value bool) *model.SlackAttachment {
	message := s.FalseResponseMessage
	if value {
		message = s.TrueResponseMessage
	}

	sa := model.SlackAttachment{
		Title:    s.Title,
		Text:     message,
		Actions:  []*model.PostAction{},
		Fallback: fmt.Sprintf("%s: %s", s.Title, message),
	}

	return &sa
}

func (s *SimpleStep) GetPropertyName() string {
	return s.PropertyName
}

func (s *SimpleStep) ShouldSkip(value bool) int {
	if value {
		return s.TrueSkip
	}

	return s.FalseSkip
}

func (s *SimpleStep) IsEmpty() bool {
	return false
}
