// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package jobs

import (
	"time"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine"
)

const ditherRenew = 50 * time.Millisecond

func NewRenewJob() RegisteredJob {
	return RegisteredJob{
		id:       "renew",
		interval: 24 * time.Hour,
		work:     runRenewJob,
	}
}

// runRenewJob calls renews the event subscription for each connected user
func runRenewJob(env engine.Env) {
	uindex, err := env.Store.LoadUserIndex()
	if err != nil {
		env.Logger.Errorf("Renew job failed to load user index. err=%v", err)
		return
	}
	env.Logger.Debugf("Renew job: %v users", len(uindex))

	for _, u := range uindex {
		asUser := engine.New(env, u.MattermostUserID)

		env.Logger.Debugf("Renewing for user: %s", u.MattermostUserID)
		_, err = asUser.RenewMyEventSubscription()
		if err != nil {
			env.Logger.Errorf("Error renewing subscription. err=%v", err)
		}

		time.Sleep(ditherRenew)
	}

	env.Logger.Debugf("Renew job finished")
}
