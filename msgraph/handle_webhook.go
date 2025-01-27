// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package msgraph

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/microcosm-cc/bluemonday"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
)

const renewSubscriptionBeforeExpiration = 12 * time.Hour

type webhook struct {
	ChangeType                     string `json:"changeType"`
	ClientState                    string `json:"clientState,omitempty"`
	Resource                       string `json:"resource,omitempty"`
	SubscriptionExpirationDateTime string `json:"subscriptionExpirationDateTime,omitempty"`
	SubscriptionID                 string `json:"subscriptionId"`
	ResourceData                   struct {
		DataType string `json:"@odata.type"`
	} `json:"resourceData"`
}

func (r *impl) HandleWebhook(w http.ResponseWriter, req *http.Request) []*remote.Notification {
	// Microsoft graph requires webhook endpoint validation, see
	// https://learn.microsoft.com/en-us/graph/change-notifications-delivery-webhooks?tabs=go#notificationurl-validation
	vtok := req.FormValue("validationToken")
	var policy = bluemonday.StrictPolicy()
	if vtok != policy.Sanitize(vtok) {
		w.WriteHeader(http.StatusBadRequest)
		r.logger.Debugf("msgraph: validation token is invalid.")
		return nil
	}

	if vtok != "" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(vtok))
		r.logger.Debugf("msgraph: validated event webhook endpoint.")
		return nil
	}

	rawData, err := io.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		r.logger.Infof("msgraph: failed to process webhook: `%v`.", err)
		return nil
	}

	// Get the list of webhooks
	var v struct {
		Value []*webhook `json:"value"`
	}
	err = json.Unmarshal(rawData, &v)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		r.logger.Infof("msgraph: failed to process webhook: `%v`.", err)
		return nil
	}

	notifications := []*remote.Notification{}
	for _, wh := range v.Value {
		n := &remote.Notification{
			SubscriptionID: wh.SubscriptionID,
			ChangeType:     wh.ChangeType,
			ClientState:    wh.ClientState,
			IsBare:         true,
			WebhookRawData: rawData,
			Webhook:        wh,
		}

		expires, err := time.Parse(time.RFC3339, wh.SubscriptionExpirationDateTime)
		if err != nil {
			r.logger.With(bot.LogContext{
				"SubscriptionID": wh.SubscriptionID,
			}).Infof("msgraph: invalid subscription expiration in webhook: `%v`.", err)
			return nil
		}
		expires = expires.Add(-renewSubscriptionBeforeExpiration)
		if time.Now().After(expires) {
			n.RecommendRenew = true
		}

		notifications = append(notifications, n)
	}

	w.WriteHeader(http.StatusAccepted)
	return notifications
}
