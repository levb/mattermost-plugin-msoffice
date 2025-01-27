// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package kvstore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
)

func encode(encrypted []byte) []byte {
	encoded := make([]byte, base64.URLEncoding.EncodedLen(len(encrypted)))
	base64.URLEncoding.Encode(encoded, encrypted)
	return encoded
}

func decode(encoded []byte) ([]byte, error) {
	decoded := make([]byte, base64.URLEncoding.DecodedLen(len(encoded)))
	n, err := base64.URLEncoding.Decode(decoded, encoded)
	if err != nil {
		return nil, err
	}
	return decoded[:n], nil
}

func encrypt(key []byte, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte(""), errors.Wrap(err, "could not create a cipher block, check key")
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return []byte(""), err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return []byte(""), err
	}

	sealed := aesgcm.Seal(nil, nonce, data, nil)
	return encode(append(nonce, sealed...)), nil
}

func decrypt(key []byte, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte(""), errors.Wrap(err, "could not create a cipher block, check key")
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return []byte(""), err
	}

	decoded, err := decode(data)
	if err != nil {
		return []byte(""), err
	}

	nonceSize := aesgcm.NonceSize()
	if len(decoded) < nonceSize {
		return []byte(""), errors.New("token too short")
	}

	nonce, encrypted := decoded[:nonceSize], decoded[nonceSize:]
	plain, err := aesgcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return []byte(""), err
	}

	return plain, nil
}

type encryptedKeyStore struct {
	store         KVStore
	encryptionKey []byte
}

var _ KVStore = (*encryptedKeyStore)(nil)

func NewEncryptedKeyStore(s KVStore, encryptionKey []byte) KVStore {
	return &encryptedKeyStore{
		store:         s,
		encryptionKey: encryptionKey,
	}
}

func (s encryptedKeyStore) Load(key string) ([]byte, error) {
	value, err := s.store.Load(key)
	if err != nil {
		return value, err
	}

	return decrypt(s.encryptionKey, value)
}

func (s encryptedKeyStore) Store(key string, data []byte) error {
	encryptedData, err := encrypt(s.encryptionKey, data)
	if err != nil {
		return errors.Wrap(err, "error encrypting data")
	}
	return s.store.Store(key, encryptedData)
}

func (s encryptedKeyStore) StoreTTL(key string, data []byte, ttlSeconds int64) error {
	encryptedData, err := encrypt(s.encryptionKey, data)
	if err != nil {
		return errors.Wrap(err, "error encrypting data")
	}

	return s.store.StoreTTL(key, encryptedData, ttlSeconds)
}

func (s encryptedKeyStore) StoreWithOptions(key string, data []byte, opts model.PluginKVSetOptions) (bool, error) {
	encryptedData, err := encrypt(s.encryptionKey, data)
	if err != nil {
		return false, errors.Wrap(err, "error encrypting data")
	}

	return s.store.StoreWithOptions(key, encryptedData, opts)
}

func (s encryptedKeyStore) Delete(key string) error {
	return s.store.Delete(key)
}
