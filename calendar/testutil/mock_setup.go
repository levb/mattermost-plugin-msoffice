// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package testutil

import (
	"github.com/stretchr/testify/mock"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

type MockPluginAPI struct {
	plugin.API
	mock.Mock
}

func (m *MockPluginAPI) KVGet(key string) ([]byte, *model.AppError) {
	args := m.Called(key)
	data, _ := args.Get(0).([]byte)
	if err := args.Get(1); err != nil {
		return nil, err.(*model.AppError)
	}
	return data, nil
}

func (m *MockPluginAPI) KVSet(key string, data []byte) *model.AppError {
	args := m.Called(key, data)
	if err := args.Get(0); err != nil {
		return err.(*model.AppError)
	}
	return nil
}

func (m *MockPluginAPI) KVSetWithExpiry(key string, data []byte, ttl int64) *model.AppError {
	args := m.Called(key, data, ttl)
	if err := args.Get(0); err != nil {
		return err.(*model.AppError)
	}
	return nil
}

func (m *MockPluginAPI) KVSetWithOptions(key string, value []byte, options model.PluginKVSetOptions) (bool, *model.AppError) {
	args := m.Called(key, value, options)
	success := args.Bool(0)
	if err := args.Get(1); err != nil {
		return success, err.(*model.AppError)
	}
	return success, nil
}

func (m *MockPluginAPI) KVDelete(key string) *model.AppError {
	args := m.Called(key)
	if err := args.Get(0); err != nil {
		return err.(*model.AppError)
	}
	return nil
}
