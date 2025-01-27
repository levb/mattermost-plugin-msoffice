// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package tz

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimeZone_GoCompatibility(t *testing.T) {
	for _, v := range windowsToIANA {
		_, err := time.LoadLocation(v)
		require.Nil(t, err)
	}
}
