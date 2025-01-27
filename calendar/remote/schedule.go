// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package remote

const (
	AvailabilityViewFree             = '0'
	AvailabilityViewTentative        = '1'
	AvailabilityViewBusy             = '2'
	AvailabilityViewOutOfOffice      = '3'
	AvailabilityViewWorkingElsewhere = '4'

	ScheduleStatusFree             = "free"
	ScheduleStatusTentative        = "tentative"
	ScheduleStatusBusy             = "busy"
	ScheduleStatusOof              = "oof"
	ScheduleStatusWorkingElsewhere = "workingElsewhere"
	ScheduleStatusUnknown          = "unknown"
)

type ScheduleInformationError struct {
	Message      string `json:"message"`
	ResponseCode string `json:"responseCode"`
}

type AvailabilityView string

// ScheduleInformation undocumented
type ScheduleInformation struct {
	// Email of user
	ScheduleID string `json:"scheduleId,omitempty"`

	// 0= free, 1= tentative, 2= busy, 3= out of office, 4= working elsewhere.
	// example "0010", which means free for first and second block, tentative for third, and free for fourth
	AvailabilityView AvailabilityView `json:"availabilityView,omitempty"`

	Error *ScheduleInformationError `json:"error"`

	ScheduleItems []*ScheduleItem `json:"scheduleItems,omitempty"`
	// WorkingHours interface{} `json:"workingHours,omitempty"`
	// Error *FreeBusyError `json:"error,omitempty"`
}

type ScheduleUserInfo struct {
	RemoteUserID string
	Mail         string
}

type ScheduleItem struct {
	Start     *DateTime
	End       *DateTime
	Status    string
	Subject   string
	Location  string
	IsPrivate bool
}
