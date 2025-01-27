// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine/mock_engine"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
)

func TestDisconnect(t *testing.T) {
	tcs := []struct {
		name           string
		command        string
		setup          func(engine.Engine)
		expectedOutput string
		expectedError  string
	}{
		{
			name:    "user not connected",
			command: "disconnect",
			setup: func(m engine.Engine) {
				mscal := m.(*mock_engine.MockEngine)
				mscal.EXPECT().GetRemoteUser("user_id").Return(&remote.User{}, store.ErrNotFound).Times(1)
			},
			expectedOutput: getNotConnectedText("http://localhost"),
			expectedError:  "",
		},
		{
			name:    "error fetching user",
			command: "disconnect",
			setup: func(m engine.Engine) {
				mscal := m.(*mock_engine.MockEngine)
				mscal.EXPECT().GetRemoteUser("user_id").Return(&remote.User{}, errors.New("some error")).Times(1)
			},
			expectedOutput: "",
			expectedError:  fmt.Sprintf("Command /%s disconnect failed: some error", config.Provider.CommandTrigger),
		},
		{
			name:    "disconnect failed",
			command: "disconnect",
			setup: func(m engine.Engine) {
				mscal := m.(*mock_engine.MockEngine)
				mscal.EXPECT().GetRemoteUser("user_id").Return(&remote.User{}, nil).Times(1)
				mscal.EXPECT().DisconnectUser("user_id").Return(errors.New("some error")).Times(1)
			},
			expectedOutput: "",
			expectedError:  fmt.Sprintf("Command /%s disconnect failed: some error", config.Provider.CommandTrigger),
		},
		{
			name:    "disconnect successful",
			command: "disconnect",
			setup: func(m engine.Engine) {
				mscal := m.(*mock_engine.MockEngine)
				mscal.EXPECT().GetRemoteUser("user_id").Return(&remote.User{}, nil).Times(1)
				mscal.EXPECT().DisconnectUser("user_id").Return(nil).Times(1)
				mscal.EXPECT().ClearSettingsPosts("user_id").Return().Times(1)
			},
			expectedOutput: "Successfully disconnected your account",
			expectedError:  "",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			conf := &config.Config{
				PluginURL: "http://localhost",
			}

			mscal := mock_engine.NewMockEngine(ctrl)
			command := Command{
				Context: &plugin.Context{},
				Args: &model.CommandArgs{
					Command: fmt.Sprintf("/%s %s", config.Provider.CommandTrigger, tc.command),
					UserId:  "user_id",
				},
				ChannelID: "channel_id",
				Config:    conf,
				Engine:    mscal,
			}

			if tc.setup != nil {
				tc.setup(mscal)
			}

			out, _, err := command.Handle()
			if tc.expectedOutput != "" {
				require.Equal(t, tc.expectedOutput, out)
			}

			if tc.expectedError != "" {
				require.Equal(t, tc.expectedError, err.Error())
			} else {
				require.Nil(t, err)
			}
		})
	}
}
