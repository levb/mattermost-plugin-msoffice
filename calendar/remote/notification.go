// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package remote

type Notification struct {
	Webhook interface{}

	// Notification data
	Subscription        *Subscription
	SubscriptionCreator *User
	Event               *Event

	// ClientState from the webhook. The handler is to validate against its own
	// persistent secret.
	ClientState string

	// Notification type
	ChangeType string

	// The (remote) subscription ID the notification is for
	SubscriptionID string

	// Remote-specific data: full raw JSON of the webhook, and the decoded
	// backend-specific struct.
	WebhookRawData []byte

	// Set if subscription renewal is recommended. The date/time logic is
	// internal to the remote implementation. The handler is to call
	// RenewSubscription() as applicable, with the appropriate user credentials.
	RecommendRenew bool

	// Set if there is no data pre-filled from processing the webhook. The
	// handler is to call GetNotificationData(), with the appropriate user
	// credentials.
	IsBare bool
}
