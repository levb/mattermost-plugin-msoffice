// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestCopy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	user := newTestUserNumbered(1)
	env, _ := makeStatusSyncTestEnv(ctrl)

	engine := New(env, user.MattermostUserID)

	engineCopy := engine.(*mscalendar).copy()

	assert.NotSame(t, engine, engineCopy)
	assert.NotSame(t, engine.(*mscalendar).actingUser, engineCopy.actingUser)
	assert.NotSame(t, engine.(*mscalendar).client, engineCopy.client)
}
