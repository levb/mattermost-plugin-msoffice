// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import "net/http"

func (api *api) getAuthorized(w http.ResponseWriter, _ *http.Request) {
	// if we've made it here, we're authorized.
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"is_authorized": true}`))
}
