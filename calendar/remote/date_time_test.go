// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package remote

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDateTime_Time(t *testing.T) {
	for _, tc := range []struct {
		DateTime
		expectedUTC          string
		expectedString       string
		expectedPrettyString string
	}{
		{
			DateTime:             DateTime{"2019-11-19T01:00:00.0000005", ""},
			expectedUTC:          "2019-11-19T01:00:00.0000005Z",
			expectedString:       "2019-11-19T01:00:00Z",
			expectedPrettyString: "19 Nov 19 01:00 UTC",
		},
		{
			DateTime:             DateTime{"2019-11-19T01:00:00.9999999", "Asia/Hong_Kong"},
			expectedUTC:          "2019-11-18T17:00:00.9999999Z",
			expectedString:       "2019-11-19T01:00:00+08:00",
			expectedPrettyString: "19 Nov 19 01:00 HKT",
		},
		{
			DateTime:             DateTime{"2019-11-19T01:00:00.9999999", "China Standard Time"},
			expectedUTC:          "2019-11-18T17:00:00.9999999Z",
			expectedString:       "2019-11-19T01:00:00+08:00",
			expectedPrettyString: "19 Nov 19 01:00 CST",
		},
		{
			DateTime:             DateTime{"2019-11-19T14:00:00.0000005", "MST"},
			expectedUTC:          "2019-11-19T21:00:00.0000005Z",
			expectedString:       "2019-11-19T14:00:00-07:00",
			expectedPrettyString: "19 Nov 19 14:00 MST",
		},
	} {
		t.Run(tc.TimeZone, func(t *testing.T) {
			require.Equal(t, tc.expectedUTC, tc.DateTime.Time().UTC().Format(time.RFC3339Nano))
			require.Equal(t, tc.expectedString, tc.DateTime.String())
			require.Equal(t, tc.expectedPrettyString, tc.DateTime.PrettyString())
		})
	}
}

func TestDateTime_In(t *testing.T) {
	for _, tc := range []struct {
		DateTime
		toTimezone           string
		expectedUTC          string
		expectedString       string
		expectedPrettyString string
	}{
		{
			DateTime:             DateTime{"2019-11-19T01:00:00.0000005", ""},
			toTimezone:           "Pacific Standard Time",
			expectedUTC:          "2019-11-19T01:00:00.0000005Z",
			expectedString:       "2019-11-18T17:00:00-08:00",
			expectedPrettyString: "18 Nov 19 17:00 PST",
		},
		{
			DateTime:             DateTime{"2019-11-19T01:00:00.9999999", "Asia/Hong_Kong"},
			toTimezone:           "Mountain Standard Time",
			expectedUTC:          "2019-11-18T17:00:00.9999999Z",
			expectedString:       "2019-11-18T10:00:00-07:00",
			expectedPrettyString: "18 Nov 19 10:00 MST",
		},
		{
			DateTime:             DateTime{"2019-11-19T01:00:00.9999999", "China Standard Time"},
			toTimezone:           "Eastern Standard Time",
			expectedUTC:          "2019-11-18T17:00:00.9999999Z",
			expectedString:       "2019-11-18T12:00:00-05:00",
			expectedPrettyString: "18 Nov 19 12:00 EST",
		},
		{
			DateTime:             DateTime{"2019-11-19T14:00:00.0000005", "MST"},
			toTimezone:           "Eastern Standard Time",
			expectedUTC:          "2019-11-19T21:00:00.0000005Z",
			expectedString:       "2019-11-19T16:00:00-05:00",
			expectedPrettyString: "19 Nov 19 22:00 EST",
		},
		{
			DateTime:             DateTime{"2019-11-19T14:00:00.0000005", "MST"},
			toTimezone:           "Central European Standard Time",
			expectedUTC:          "2019-11-19T21:00:00.0000005Z",
			expectedString:       "2019-11-19T22:00:00+01:00",
			expectedPrettyString: "18 Nov 19 17:00 CEST",
		},
	} {
		t.Run(tc.toTimezone, func(t *testing.T) {
			dt := tc.DateTime.In(tc.toTimezone)
			require.Equal(t, tc.expectedUTC, dt.Time().UTC().Format(time.RFC3339Nano))
			require.Equal(t, tc.expectedString, dt.String())
			require.Equal(t, tc.toTimezone, dt.TimeZone)
		})
	}
}
