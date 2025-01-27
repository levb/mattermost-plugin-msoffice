// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package store

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/testutil"
)

func TestLoadUserWelcomePost(t *testing.T) {
	mockAPI, store, _, _, _ := GetMockSetup(t)

	tests := []struct {
		name       string
		setup      func(*testutil.MockPluginAPI)
		assertions func(*testing.T, string, error)
	}{
		{
			name: "Error loading user welcome post",
			setup: func(mockAPI *testutil.MockPluginAPI) {
				mockAPI.On("KVGet", mock.AnythingOfType("string")).Return(nil, &model.AppError{Message: "KVGet failed"})
			},
			assertions: func(t *testing.T, resp string, err error) {
				require.Equal(t, resp, "")
				require.EqualError(t, err, "failed plugin KVGet: KVGet failed")
			},
		},
		{
			name: "Success loading user welcome post",
			setup: func(mockAPI *testutil.MockPluginAPI) {
				mockAPI.On("KVGet", mock.AnythingOfType("string")).Return([]byte(`"mockPostID"`), nil)
			},
			assertions: func(t *testing.T, resp string, err error) {
				require.NoError(t, err)
				require.Equal(t, "mockPostID", resp)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI.ExpectedCalls = nil
			tt.setup(mockAPI)

			resp, err := store.LoadUserWelcomePost("mockMMUserID")

			tt.assertions(t, resp, err)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestStoreUserWelcomePost(t *testing.T) {
	mockAPI, store, _, _, _ := GetMockSetup(t)

	tests := []struct {
		name       string
		setup      func(*testutil.MockPluginAPI)
		assertions func(*testing.T, error)
	}{
		{
			name: "Error storing user welcome post",
			setup: func(mockAPI *testutil.MockPluginAPI) {
				mockAPI.On("KVSet", mock.Anything, mock.Anything).Return(&model.AppError{Message: "KVSet failed"})
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContainsf(t, err, "failed plugin KVSet (ttl: 0s)", `"mockMMUserID": KVSet failed`)
			},
		},
		{
			name: "Success storing user welcome post",
			setup: func(mockAPI *testutil.MockPluginAPI) {
				mockAPI.On("KVSet", mock.Anything, mock.Anything).Return(nil)
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI.ExpectedCalls = nil
			tt.setup(mockAPI)

			err := store.StoreUserWelcomePost("mockMMUserID", "mockPostID")

			tt.assertions(t, err)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestDeleteUserWelcomePost(t *testing.T) {
	mockAPI, store, _, _, _ := GetMockSetup(t)

	tests := []struct {
		name       string
		setup      func(*testutil.MockPluginAPI)
		assertions func(*testing.T, string, error)
	}{
		{
			name: "Error deleting user welcome post",
			setup: func(mockAPI *testutil.MockPluginAPI) {
				mockAPI.On("KVGet", mock.AnythingOfType("string")).Return([]byte(`"mockPostID"`), nil)
				mockAPI.On("KVDelete", mock.Anything, mock.Anything).Return(&model.AppError{Message: "KVDelete failed"})
			},
			assertions: func(t *testing.T, resp string, err error) {
				require.Equal(t, "", resp)
				require.ErrorContains(t, err, "failed plugin KVdelete", "KVDelete failed")
			},
		},
		{
			name: "Success deleting user welcome post",
			setup: func(mockAPI *testutil.MockPluginAPI) {
				mockAPI.On("KVGet", mock.AnythingOfType("string")).Return([]byte(`"mockPostID"`), nil)
				mockAPI.On("KVDelete", mock.Anything, mock.Anything).Return(nil)
			},
			assertions: func(t *testing.T, resp string, err error) {
				require.Equal(t, "mockPostID", resp)
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI.ExpectedCalls = nil
			tt.setup(mockAPI)

			resp, err := store.DeleteUserWelcomePost("mockMMUserID")

			tt.assertions(t, resp, err)

			mockAPI.AssertExpectations(t)
		})
	}
}
