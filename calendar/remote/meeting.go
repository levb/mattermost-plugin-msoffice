// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package remote

import "time"

type FindMeetingTimesParameters struct {
	ReturnSuggestionReasons   *bool               `json:"returnSuggestionReasons,omitempty"`
	LocationConstraint        *LocationConstraint `json:"locationConstraint,omitempty"`
	TimeConstraint            *TimeConstraint     `json:"timeConstraint,omitempty"`
	MeetingDuration           *time.Duration      `json:"meetingDuration,omitempty"`
	MaxCandidates             *int                `json:"maxCandidates,omitempty"`
	IsOrganizerOptional       *bool               `json:"isOrganizerOptional,omitempty"`
	MinimumAttendeePercentage *float64            `json:"minimumAttendeePercentage,omitempty"`
	Attendees                 []Attendee          `json:"attendees,omitempty"`
}

type TimeConstraint struct {
	ActivityDomain string     `json:"activityDomain,omitempty"`
	TimeSlots      []TimeSlot `json:"timeSlots,omitempty"`
}
type MeetingTimeSuggestion struct {
	MeetingTimeSlot       *TimeSlot
	SuggestionReason      string `json:"suggestionReason"`
	OrganizerAvailability string `json:"organizerAvailability"`
	Locations             []*Location
	AttendeeAvailability  []*AttendeeAvailability
	Confidence            float32 `json:"confidence"`
	Order                 int32   `json:"order"`
}

type AttendeeAvailability struct {
	Attendee     *Attendee
	Availability string `json:"availability"`
}

type MeetingTimeSuggestionResults struct {
	EmptySuggestionReason  string                   `json:"emptySuggestionReason"`
	MeetingTimeSuggestions []*MeetingTimeSuggestion `json:"meetingTimeSuggestions"`
}

type TimeSlot struct {
	Start *DateTime `json:"start,omitempty"`
	End   *DateTime `json:"end,omitempty"`
}

type LocationConstraint struct {
	IsRequired      *bool                    `json:"isRequired,omitempty"`
	SuggestLocation *bool                    `json:"suggestLocation,omitempty"`
	Locations       []LocationConstraintItem `json:"locations,omitempty"`
}

type LocationConstraintItem struct {
	Location            *Location
	ResolveAvailability *bool `json:"resolveAvailability,omitempty"`
}
