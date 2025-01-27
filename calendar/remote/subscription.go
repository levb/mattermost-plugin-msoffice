// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package remote

type Subscription struct {
	ID                 string `json:"id"`
	ResourceID         string `json:"resourceId,omitempty"`
	Resource           string `json:"resource,omitempty"`
	ApplicationID      string `json:"applicationId,omitempty"`
	ChangeType         string `json:"changeType,omitempty"`
	ClientState        string `json:"clientState,omitempty"`
	NotificationURL    string `json:"notificationUrl,omitempty"`
	ExpirationDateTime string `json:"expirationDateTime,omitempty"`
	CreatorID          string `json:"creatorId,omitempty"`
}
