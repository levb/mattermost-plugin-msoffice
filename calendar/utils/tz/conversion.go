// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package tz

import (
	"time"
)

// Go converts a time zone into a Go-compatible time zone if it is not compatible.
func Go(timeZone string) string {
	_, err := time.LoadLocation(timeZone)
	if err == nil {
		return timeZone
	}

	return windowsToIANA[timeZone]
}

// Microsoft converts a time zone into a Microsoft-compatible time zone if it is not compatible.
func Microsoft(timeZone string) string {
	_, ok := iANAToWindows[timeZone]
	if ok {
		return timeZone
	}

	return timeZone
}
