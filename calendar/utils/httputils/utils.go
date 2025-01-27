// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package httputils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/pkg/errors"
)

func NormalizeRemoteBaseURL(mattermostSiteURL, remoteURL string) (string, error) {
	u, err := url.Parse(remoteURL)
	if err != nil {
		return "", err
	}
	if u.Host == "" {
		ss := strings.Split(u.Path, "/")
		if len(ss) > 0 && ss[0] != "" {
			u.Host = ss[0]
			u.Path = path.Join(ss[1:]...)
		}
		u, err = url.Parse(u.String())
		if err != nil {
			return "", err
		}
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid URL, no hostname: %q", remoteURL)
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}

	remoteURL = strings.TrimSuffix(u.String(), "/")
	if remoteURL == strings.TrimSuffix(mattermostSiteURL, "/") {
		return "", fmt.Errorf("%s is the Mattermost site URL. Please use the remote application's URL", remoteURL)
	}

	return remoteURL, nil
}

func WriteJSONError(w http.ResponseWriter, statusCode int, summary string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	b, _ := json.Marshal(struct {
		Error   string `json:"error"`
		Summary string `json:"details"`
	}{
		Summary: summary,
		Error:   err.Error(),
	})
	_, _ = w.Write(b)
}

func WriteInternalServerError(w http.ResponseWriter, err error) {
	WriteJSONError(w, http.StatusInternalServerError, "An internal error has occurred. Check app server logs for details.", err)
}

func WriteBadRequestError(w http.ResponseWriter, err error) {
	WriteJSONError(w, http.StatusBadRequest, "Invalid request.", err)
}

func WriteNotFoundError(w http.ResponseWriter, err error) {
	WriteJSONError(w, http.StatusNotFound, "Not found.", err)
}

func WriteUnauthorizedError(w http.ResponseWriter, err error) {
	WriteJSONError(w, http.StatusUnauthorized, "Unauthorized.", err)
}

func WriteJSONResponse(w http.ResponseWriter, data any, statusCode int) error {
	jsonResponse, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "couldn't parse response")
	}

	w.WriteHeader(statusCode)
	if _, err := w.Write(jsonResponse); err != nil {
		return errors.Wrap(err, "couldn't send response to user")
	}

	return nil
}
