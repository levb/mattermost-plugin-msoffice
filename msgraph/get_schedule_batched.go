// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package msgraph

import (
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
)

type getScheduleResponse struct {
	Error *remote.APIError              `json:"error,omitempty"`
	Value []*remote.ScheduleInformation `json:"value,omitempty"`
}

type getScheduleSingleResponse struct {
	Headers map[string]string   `json:"headers"`
	ID      string              `json:"id"`
	Body    getScheduleResponse `json:"body"`
	Status  int                 `json:"status"`
}

type getScheduleBatchResponse struct {
	Responses []*getScheduleSingleResponse `json:"responses"`
}

type getScheduleRequestParams struct {
	// Overall start and end of entire search window
	StartTime *remote.DateTime `json:"startTime"`
	EndTime   *remote.DateTime `json:"endTime"`

	// List of emails of users that we want to check
	Schedules []string `json:"schedules"`

	/*
		Size of each chunk of time we want to check
		This can be equal to end - start if we want, or we can get more granular results by making it shorter.
		For the graph API: The default is 30 minutes, minimum is 6, maximum is 1440
		15 is currently being used on our end
	*/
	AvailabilityViewInterval int `json:"availabilityViewInterval"`
}

func (c *client) GetSchedule(requests []*remote.ScheduleUserInfo, startTime, endTime *remote.DateTime, availabilityViewInterval int) ([]*remote.ScheduleInformation, error) {
	params := &getScheduleRequestParams{
		StartTime:                startTime,
		EndTime:                  endTime,
		AvailabilityViewInterval: availabilityViewInterval,
	}

	allRequests := []*singleRequest{}
	for _, req := range requests {
		allRequests = append(allRequests, makeSingleRequestForGetSchedule(req, params))
	}
	batchRequests := prepareBatchRequests(allRequests)

	var batchResponses []*getScheduleBatchResponse

	for _, req := range batchRequests {
		res := &getScheduleBatchResponse{}
		err := c.batchRequest(req, res)
		if err != nil {
			return nil, errors.Wrap(err, "msgraph batch GetSchedule")
		}

		batchResponses = append(batchResponses, res)
	}

	result := []*remote.ScheduleInformation{}
	for _, batchRes := range batchResponses {
		for _, r := range batchRes.Responses {
			if r.Body.Error == nil {
				result = append(result, r.Body.Value...)
			} else {
				c.Warnf("Failed to process schedule. err=%s", r.Body.Error.Message)
			}
		}
	}

	return result, nil
}

func makeSingleRequestForGetSchedule(request *remote.ScheduleUserInfo, params *getScheduleRequestParams) *singleRequest {
	u := "/Users/" + url.PathEscape(request.RemoteUserID) + "/getSchedule"
	req := &singleRequest{
		URL:    u,
		Method: http.MethodPost,
		ID:     request.RemoteUserID,
		Body: &getScheduleRequestParams{
			Schedules:                []string{request.Mail},
			StartTime:                params.StartTime,
			EndTime:                  params.EndTime,
			AvailabilityViewInterval: params.AvailabilityViewInterval,
		},
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
	return req
}
