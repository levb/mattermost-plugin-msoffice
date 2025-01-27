// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package bot

import "github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/flow"

type FlowController interface {
	Start(userID string) error
	NextStep(userID string, from int, value bool) error
	Cancel(userID string) error
}

func (bot *bot) Start(userID string) error {
	err := bot.setFlowStep(userID, 0)
	if err != nil {
		return err
	}
	return bot.processStep(userID, bot.flow.Step(0), 0)
}

func (bot *bot) NextStep(userID string, from int, value bool) error {
	step, err := bot.getFlowStep(userID)
	if err != nil {
		return err
	}

	if step != from {
		return nil
	}

	skip := bot.flow.Step(step).ShouldSkip(value)
	step += 1 + skip
	if step >= bot.flow.Length() {
		bot.removeFlowStep(userID)
		bot.flow.FlowDone(userID)
		return nil
	}

	err = bot.setFlowStep(userID, step)
	if err != nil {
		return err
	}

	return bot.processStep(userID, bot.flow.Step(step), step)
}

func (bot *bot) Cancel(userID string) error {
	stepIndex, err := bot.getFlowStep(userID)
	if err != nil {
		return err
	}

	step := bot.flow.Step(stepIndex)
	if step == nil {
		return nil
	}

	postID, err := bot.flowStore.GetPostID(userID, step.GetPropertyName())
	if err != nil {
		return err
	}

	err = bot.DeletePost(postID)
	if err != nil {
		return err
	}

	return nil
}

func (bot *bot) setFlowStep(userID string, step int) error {
	return bot.flowStore.SetCurrentStep(userID, step)
}

func (bot *bot) getFlowStep(userID string) (int, error) {
	return bot.flowStore.GetCurrentStep(userID)
}

func (bot *bot) removeFlowStep(userID string) error {
	return bot.flowStore.DeleteCurrentStep(userID)
}

func (bot *bot) processStep(userID string, step flow.Step, i int) error {
	if step == nil {
		bot.Errorf("Step nil")
	}

	if bot.flow == nil {
		bot.Errorf("Bot nil")
	}

	if bot.flowStore == nil {
		bot.Errorf("Store nil")
	}
	postID, err := bot.DMWithAttachments(userID, step.PostSlackAttachment(bot.pluginURL+bot.flow.URL(), i))
	if err != nil {
		return err
	}

	if step.IsEmpty() {
		return bot.NextStep(userID, i, false)
	}

	err = bot.flowStore.SetPostID(userID, step.GetPropertyName(), postID)
	if err != nil {
		return err
	}

	return nil
}
