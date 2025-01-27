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
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
)

func TestDailySummary(t *testing.T) {
	testcase := []struct {
		name       string
		parameters []string
		setup      func(engine.Engine)
		assertions func(t *testing.T, output string, err error)
	}{
		{
			name:       "no parameters",
			parameters: []string{},
			setup:      func(_ engine.Engine) {},
			assertions: func(t *testing.T, output string, err error) {
				require.Equal(t, getDailySummaryHelp(), output)
				require.Nil(t, err)
			},
		},
		{
			name:       "view today's summary",
			parameters: []string{"view"},
			setup: func(m engine.Engine) {
				mscal := m.(*mock_engine.MockEngine)
				mscal.EXPECT().GetDaySummaryForUser(gomock.Any(), gomock.Any()).Return("Today's Summary", nil).Times(1)
			},
			assertions: func(t *testing.T, output string, err error) {
				require.Equal(t, "Today's Summary", output)
				require.Nil(t, err)
			},
		},
		{
			name:       "view tomorrow's summary",
			parameters: []string{"tomorrow"},
			setup: func(m engine.Engine) {
				mscal := m.(*mock_engine.MockEngine)
				mscal.EXPECT().GetDaySummaryForUser(gomock.Any(), gomock.Any()).Return("Tomorrow's Summary", nil).Times(1)
			},
			assertions: func(t *testing.T, output string, err error) {
				require.Equal(t, "Tomorrow's Summary", output)
				require.Nil(t, err)
			},
		},
		{
			name:       "error viewing summary",
			parameters: []string{"view"},
			setup: func(m engine.Engine) {
				mscal := m.(*mock_engine.MockEngine)
				mscal.EXPECT().GetDaySummaryForUser(gomock.Any(), gomock.Any()).Return("", errors.New("summary error")).Times(1)
			},
			assertions: func(t *testing.T, output string, err error) {
				require.Equal(t, "summary error", output)
				require.Equal(t, "summary error", err.Error())
			},
		},
		{
			name:       "set time with invalid parameter count",
			parameters: []string{"time"},
			setup:      func(_ engine.Engine) {},
			assertions: func(t *testing.T, output string, err error) {
				require.Equal(t, getDailySummarySetTimeErrorMessage(), output)
				require.Nil(t, err)
			},
		},
		{
			name:       "set time successfully",
			parameters: []string{"time", "09:00"},
			setup: func(m engine.Engine) {
				mscal := m.(*mock_engine.MockEngine)
				mscal.EXPECT().SetDailySummaryPostTime(gomock.Any(), "09:00").Return(&store.DailySummaryUserSettings{PostTime: "09:00", Timezone: "UTC", Enable: true}, nil).Times(1)
			},
			assertions: func(t *testing.T, output string, err error) {
				require.Equal(t, dailySummaryResponse(&store.DailySummaryUserSettings{PostTime: "09:00", Timezone: "UTC", Enable: true}), output)
				require.Nil(t, err)
			},
		},
		{
			name:       "error setting time",
			parameters: []string{"time", "09:00"},
			setup: func(m engine.Engine) {
				mscal := m.(*mock_engine.MockEngine)
				mscal.EXPECT().SetDailySummaryPostTime(gomock.Any(), "09:00").Return(nil, errors.New("time error")).Times(1)
			},
			assertions: func(t *testing.T, output string, err error) {
				require.Equal(t, "time error\n"+getDailySummarySetTimeErrorMessage(), output)
				require.Nil(t, err)
			},
		},
		{
			name:       "get settings when not configured",
			parameters: []string{"settings"},
			setup: func(m engine.Engine) {
				mscal := m.(*mock_engine.MockEngine)
				mscal.EXPECT().GetDailySummarySettingsForUser(gomock.Any()).Return(&store.DailySummaryUserSettings{}, nil).Times(1)
			},
			assertions: func(t *testing.T, output string, err error) {
				require.Equal(t, "Your daily summary time is not yet configured.\n"+getDailySummarySetTimeErrorMessage(), output)
				require.Nil(t, err)
			},
		},
		{
			name:       "get settings when configured but disabled",
			parameters: []string{"settings"},
			setup: func(m engine.Engine) {
				mscal := m.(*mock_engine.MockEngine)
				mscal.EXPECT().GetDailySummarySettingsForUser(gomock.Any()).Return(&store.DailySummaryUserSettings{PostTime: "09:00", Timezone: "UTC", Enable: false}, nil).Times(1)
			},
			assertions: func(t *testing.T, output string, err error) {
				require.Equal(t, "Your daily summary is configured to show at 09:00 UTC, but is disabled. Enable it with `/ summary enable`.", output)
				require.Nil(t, err)
			},
		},
		{
			name:       "get settings when configured and enabled",
			parameters: []string{"settings"},
			setup: func(m engine.Engine) {
				mscal := m.(*mock_engine.MockEngine)
				mscal.EXPECT().GetDailySummarySettingsForUser(gomock.Any()).Return(&store.DailySummaryUserSettings{PostTime: "09:00", Timezone: "UTC", Enable: true}, nil).Times(1)
			},
			assertions: func(t *testing.T, output string, err error) {
				require.Equal(t, "Your daily summary is configured to show at 09:00 UTC.", output)
				require.Nil(t, err)
			},
		},
		{
			name:       "error getting settings",
			parameters: []string{"settings"},
			setup: func(m engine.Engine) {
				mscal := m.(*mock_engine.MockEngine)
				mscal.EXPECT().GetDailySummarySettingsForUser(gomock.Any()).Return(nil, errors.New("settings error")).Times(1)
			},
			assertions: func(t *testing.T, output string, err error) {
				require.Equal(t, "settings error\nYou may need to configure your daily summary using the commands below.\n"+getDailySummaryHelp(), output)
				require.Nil(t, err)
			},
		},
		{
			name:       "enable daily summary",
			parameters: []string{"enable"},
			setup: func(m engine.Engine) {
				mscal := m.(*mock_engine.MockEngine)
				mscal.EXPECT().SetDailySummaryEnabled(gomock.Any(), true).Return(&store.DailySummaryUserSettings{PostTime: "09:00", Timezone: "UTC", Enable: true}, nil).Times(1)
			},
			assertions: func(t *testing.T, output string, err error) {
				require.Equal(t, dailySummaryResponse(&store.DailySummaryUserSettings{PostTime: "09:00", Timezone: "UTC", Enable: true}), output)
				require.Nil(t, err)
			},
		},
		{
			name:       "disable daily summary",
			parameters: []string{"disable"},
			setup: func(m engine.Engine) {
				mscal := m.(*mock_engine.MockEngine)
				mscal.EXPECT().SetDailySummaryEnabled(gomock.Any(), false).Return(&store.DailySummaryUserSettings{PostTime: "09:00", Timezone: "UTC", Enable: false}, nil).Times(1)
			},
			assertions: func(t *testing.T, output string, err error) {
				require.Equal(t, dailySummaryResponse(&store.DailySummaryUserSettings{PostTime: "09:00", Timezone: "UTC", Enable: false}), output)
				require.Nil(t, err)
			},
		},
		{
			name:       "invalid command",
			parameters: []string{"invalid"},
			setup:      func(_ engine.Engine) {},
			assertions: func(t *testing.T, output string, err error) {
				require.Equal(t, "Invalid command. Please try again\n\n"+getDailySummaryHelp(), output)
				require.Nil(t, err)
			},
		},
	}
	for _, tt := range testcase {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			conf := &config.Config{
				PluginURL: "http://localhost",
			}

			mscal := mock_engine.NewMockEngine(ctrl)
			command := Command{
				Context: &plugin.Context{},
				Args: &model.CommandArgs{
					Command: fmt.Sprintf("/%s dailySummary", config.Provider.CommandTrigger),
					UserId:  "mockUserID",
				},
				ChannelID: "mockChannelID",
				Config:    conf,
				Engine:    mscal,
			}

			tt.setup(mscal)

			out, _, err := command.dailySummary(tt.parameters...)

			tt.assertions(t, out, err)
		})
	}
}
