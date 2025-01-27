// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package jobs

import "github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine"

// Unique id for the status sync job
const statusSyncJobID = "status_sync"

// NewStatusSyncJob creates a RegisteredJob with the parameters specific to the StatusSyncJob
func NewStatusSyncJob() RegisteredJob {
	return RegisteredJob{
		id:       statusSyncJobID,
		interval: engine.StatusSyncJobInterval,
		work:     runSyncJob,
	}
}

// runSyncJob synchronizes all users' statuses between mscalendar and Mattermost.
func runSyncJob(env engine.Env) {
	env.Logger.Debugf("User status sync job beginning")

	_, syncJobSummary, err := engine.New(env, "").SyncAll()
	if err != nil {
		env.Logger.Errorf("Error during user status sync job. err=%v", err)
	}

	env.Logger.Debugf("User status sync job finished.\nSummary\nNumber of users processed:- %d\nNumber of users had their status changed:- %d\nNumber of users had errors:- %d", syncJobSummary.NumberOfUsersProcessed, syncJobSummary.NumberOfUsersStatusChanged, syncJobSummary.NumberOfUsersFailedStatusChanged)
}
