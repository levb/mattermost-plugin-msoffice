// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package msgraph

import (
	"net/http"
)

const maxNumRequestsPerBatch = 20

type singleRequest struct {
	Body    interface{}       `json:"body"`
	Headers map[string]string `json:"headers"`
	ID      string            `json:"id"`
	URL     string            `json:"url"`
	Method  string            `json:"method"`
}

type fullBatchRequest struct {
	Requests []*singleRequest `json:"requests"`
}

func (c *client) batchRequest(req fullBatchRequest, out interface{}) error {
	u := "https://graph.microsoft.com/v1.0/$batch"

	_, err := c.CallJSON(http.MethodPost, u, req, out)
	return err
}

func prepareBatchRequests(requests []*singleRequest) []fullBatchRequest {
	numFullRequests := len(requests) / maxNumRequestsPerBatch
	if len(requests)%maxNumRequestsPerBatch != 0 {
		numFullRequests++
	}

	result := []fullBatchRequest{}

	for i := 0; i < numFullRequests; i++ {
		startIdx := i * maxNumRequestsPerBatch
		endIdx := startIdx + maxNumRequestsPerBatch
		if i == numFullRequests-1 {
			endIdx = len(requests)
		}

		slice := requests[startIdx:endIdx]
		batchReq := fullBatchRequest{Requests: slice}
		result = append(result, batchReq)
	}

	return result
}
