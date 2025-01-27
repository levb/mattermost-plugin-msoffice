// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package kvstore

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
)

type KVStore interface {
	Load(key string) ([]byte, error)
	Store(key string, data []byte) error
	StoreTTL(key string, data []byte, ttlSeconds int64) error
	StoreWithOptions(key string, value []byte, opts model.PluginKVSetOptions) (bool, error)
	Delete(key string) error
}

var ErrNotFound = errors.New("not found")

const (
	atomicRetryLimit = 5
	atomicRetryWait  = 30 * time.Millisecond
)

func Ensure(s KVStore, key string, newValue []byte) ([]byte, error) {
	value, err := s.Load(key)
	switch err {
	case nil:
		return value, nil
	case ErrNotFound:
		break
	default:
		return nil, err
	}

	err = s.Store(key, newValue)
	if err != nil {
		return nil, err
	}

	// Load again in case we lost the race to another server
	value, err = s.Load(key)
	if err != nil {
		return newValue, nil
	}
	return value, nil
}

func LoadJSON(s KVStore, key string, v interface{}) (returnErr error) {
	data, err := s.Load(key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func StoreJSON(s KVStore, key string, v interface{}) (returnErr error) {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.Store(key, data)
}

func AtomicModifyWithOptions(s KVStore, key string, modify func(initialValue []byte, storeErr error) ([]byte, *model.PluginKVSetOptions, error)) error {
	currentAttempt := 0
	for {
		initialBytes, appErr := s.Load(key)
		newValue, opts, err := modify(initialBytes, appErr)
		if err != nil {
			return errors.Wrap(err, "modification error")
		}

		// No modifications have been done. No reason to hit the plugin API.
		if bytes.Equal(initialBytes, newValue) {
			return nil
		}

		if opts == nil {
			opts = &model.PluginKVSetOptions{}
		}
		opts.Atomic = true
		opts.OldValue = initialBytes
		success, setError := s.StoreWithOptions(key, newValue, *opts)
		if setError != nil {
			return errors.Wrap(setError, "problem writing value")
		}
		if success {
			return nil
		}

		currentAttempt++
		if currentAttempt >= atomicRetryLimit {
			return errors.New("reached write attempt limit")
		}

		time.Sleep(atomicRetryWait)
	}
}

func AtomicModify(s KVStore, key string, modify func(initialValue []byte, storeErr error) ([]byte, error)) error {
	return AtomicModifyWithOptions(s, key, func(initialValue []byte, storeErr error) ([]byte, *model.PluginKVSetOptions, error) {
		b, err := modify(initialValue, storeErr)
		return b, nil, err
	})
}
