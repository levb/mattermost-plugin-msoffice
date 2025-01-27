// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package remote

import (
	"time"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/tz"
)

type EmailAddress struct {
	Address string `json:"address"`
	Name    string `json:"name,omitempty"`
}
type DateTime struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone,omitempty"`
}

const (
	RFC3339NanoNoTimezone = "2006-01-02T15:04:05.999999999"
	UndefinedDatetime     = "n/a"
)

// NewDateTime creates a DateTime that is compatible with Microsoft's API.
func NewDateTime(t time.Time, timeZone string) *DateTime {
	timeZone = tz.Microsoft(timeZone)

	return &DateTime{
		DateTime: t.Format(RFC3339NanoNoTimezone),
		TimeZone: timeZone,
	}
}

// String returns the RFC3339 representation of the provided datetime. UndefinedDatetime if not set.
func (dt DateTime) String() string {
	t := dt.Time()
	if t.IsZero() {
		return UndefinedDatetime
	}
	return t.Format(time.RFC3339)
}

func (dt DateTime) PrettyString() string {
	t := dt.Time()
	if t.IsZero() {
		return UndefinedDatetime
	}
	return t.Format(time.RFC822)
}

func (dt DateTime) In(timeZone string) *DateTime {
	t := dt.Time()
	if t.IsZero() {
		return &dt
	}

	loc, err := time.LoadLocation(tz.Go(timeZone))
	if err == nil {
		t = t.In(loc)
	}

	return &DateTime{
		TimeZone: timeZone,
		DateTime: t.Format(RFC3339NanoNoTimezone),
	}
}

func (dt DateTime) Time() time.Time {
	loc, err := time.LoadLocation(tz.Go(dt.TimeZone))
	if err != nil {
		return time.Time{}
	}

	t, err := time.ParseInLocation(RFC3339NanoNoTimezone, dt.DateTime, loc)
	if err != nil {
		return time.Time{}
	}

	return t
}
