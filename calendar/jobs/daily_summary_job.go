// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package jobs

import (
	"time"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine"
)

// Unique id for the daily summary job
const dailySummaryJobID = "daily_summary"

const dailySummaryJobInterval = engine.DailySummaryJobInterval

// NewDailySummaryJob creates a RegisteredJob with the parameters specific to the DailySummaryJob
func NewDailySummaryJob() RegisteredJob {
	return RegisteredJob{
		id:       dailySummaryJobID,
		interval: dailySummaryJobInterval,
		work:     runDailySummaryJob,
	}
}

// runDailySummaryJob delivers the daily calendar summary to all users who have their settings configured to receive it now
func runDailySummaryJob(env engine.Env) {
	env.Logger.Debugf("Daily summary job beginning")

	err := engine.New(env, "").ProcessAllDailySummary(time.Now())
	if err != nil {
		env.Logger.Errorf("Error during daily summary job. err=%v", err)
	}

	env.Logger.Debugf("Daily summary job finished")
}
