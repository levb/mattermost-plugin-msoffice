// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package msgraph

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
)

func TestMakeSingleRequestForGetSchedule(t *testing.T) {
	start := time.Now().UTC()
	end := time.Now().UTC().Add(20)
	params := &getScheduleRequestParams{
		StartTime:                remote.NewDateTime(start, "UTC"),
		EndTime:                  remote.NewDateTime(end, "UTC"),
		AvailabilityViewInterval: 15,
	}
	req := &remote.ScheduleUserInfo{
		RemoteUserID: "remote_user_id",
		Mail:         "mail@example.com",
	}

	out := makeSingleRequestForGetSchedule(req, params)
	require.Equal(t, "/Users/remote_user_id/getSchedule", out.URL)
	require.Equal(t, "POST", out.Method)
	require.Equal(t, 1, len(out.Headers))
	require.Equal(t, "application/json", out.Headers["Content-Type"])

	body := out.Body.(*getScheduleRequestParams)
	require.Equal(t, params.StartTime.String(), body.StartTime.String())
	require.Equal(t, params.EndTime.String(), body.EndTime.String())
	require.Equal(t, 15, body.AvailabilityViewInterval)
	require.Equal(t, "mail@example.com", body.Schedules[0])
}
