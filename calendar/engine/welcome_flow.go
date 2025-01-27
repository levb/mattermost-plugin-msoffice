// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"fmt"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/flow"
)

type WelcomeFlow struct {
	controller       bot.FlowController
	onFlowDone       func(userID string)
	url              string
	providerFeatures config.ProviderFeatures
	steps            []flow.Step
}

func NewWelcomeFlow(bot bot.FlowController, welcomer Welcomer, providerFeatures config.ProviderFeatures) *WelcomeFlow {
	wf := WelcomeFlow{
		url:              "/welcome",
		controller:       bot,
		onFlowDone:       welcomer.WelcomeFlowEnd,
		providerFeatures: providerFeatures,
	}
	wf.makeSteps()
	return &wf
}

func (wf *WelcomeFlow) Step(i int) flow.Step {
	if i < 0 {
		return nil
	}
	if i >= len(wf.steps) {
		return nil
	}
	return wf.steps[i]
}

func (wf *WelcomeFlow) URL() string {
	return wf.url
}

func (wf *WelcomeFlow) Length() int {
	return len(wf.steps)
}

func (wf *WelcomeFlow) StepDone(userID string, step int, value bool) {
	wf.controller.NextStep(userID, step, value)
}

func (wf *WelcomeFlow) FlowDone(userID string) {
	wf.onFlowDone(userID)
}

func (wf *WelcomeFlow) makeSteps() {
	steps := []flow.Step{
		&flow.EmptyStep{
			Title:   "Update Status",
			Message: fmt.Sprintf("You can type `/%s` to configure the plugin to update your status to \"Away\" or \"Do not disturb\" when you're in a meeting.", config.Provider.CommandTrigger),
		},
		&flow.SimpleStep{
			Title:                "Set Custom Status",
			Message:              "Do you want to set a Mattermost custom status automatically when you're in a meeting?",
			PropertyName:         store.SetCustomStatusPropertyName,
			TrueButtonMessage:    "Yes - set my Mattermost custom status to :calendar: automatically",
			FalseButtonMessage:   "No, don't set a custom status",
			TrueResponseMessage:  "We'll set a Mattermost custom status automatically when you're in a meeting.",
			FalseResponseMessage: "We won't set a Mattermost custom status when you're in a meeting.",
		},
		// &flow.SimpleStep{
		// 	Title:                "Confirm status change",
		// 	Message:              "Do you want to receive confirmations before we update your status for each event?",
		// 	PropertyName:         store.GetConfirmationPropertyName,
		// 	TrueButtonMessage:    "Yes - I would like to get confirmations",
		// 	FalseButtonMessage:   "No - Update my status automatically",
		// 	TrueResponseMessage:  "Cool, we'll also send you confirmations before updating your status.",
		// 	FalseResponseMessage: "Cool, we'll update your status automatically with no confirmation.",
		// },
		// &flow.SimpleStep{
		// 	Title:                "Status during meetings",
		// 	Message:              "Do you want to set your status to `Away` or to `Do not Disturb` while you are on a meeting? Setting to `Do Not Disturb` will silence notifications.",
		// 	PropertyName:         store.ReceiveNotificationsDuringMeetingName,
		// 	TrueButtonMessage:    "Away",
		// 	FalseButtonMessage:   "Do not Disturb",
		// 	TrueResponseMessage:  "Great, your status will be set to Away.",
		// 	FalseResponseMessage: "Great, your status will be set to Do not Disturb.",
		// },
	}

	if wf.providerFeatures.EventNotifications {
		steps = append(steps, &flow.SimpleStep{
			Title:                "Subscribe to events",
			Message:              "Do you want to receive notifications when you are invited to an event?",
			PropertyName:         store.SubscribePropertyName,
			TrueButtonMessage:    "Yes - I would like to receive notifications for new events",
			FalseButtonMessage:   "No - Do not notify me of new events",
			TrueResponseMessage:  "Great, you will receive a message any time you receive a new event.",
			FalseResponseMessage: "Great, you will not receive any notification on new events.",
		})
	}

	steps = append(steps, &flow.SimpleStep{
		Title:                "Receive reminder",
		Message:              "Do you want to receive a reminder for upcoming events?",
		PropertyName:         store.ReceiveUpcomingEventReminderName,
		TrueButtonMessage:    "Yes - I would like to receive reminders for upcoming events",
		FalseButtonMessage:   "No - Do not notify me of upcoming events",
		TrueResponseMessage:  "Great, you will receive a message before your meetings.",
		FalseResponseMessage: "Great, you will not receive any notification for upcoming events.",
	}, &flow.EmptyStep{
		Title:   "Daily Summary",
		Message: fmt.Sprintf("Remember that you can set-up a daily summary by typing `/%s summary time 8:00AM` or using `/%s settings` to access the settings.", config.Provider.CommandTrigger, config.Provider.CommandTrigger),
	})

	wf.steps = steps
}
