/**
 * Copyright 2019 Rightech IoT. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package state

import (
	"errors"
	"strings"
	"sync"

	"github.com/etcd-io/bbolt"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/objx"
)

type DB interface {
	Update(func(tx *bbolt.Tx) error) error
	View(func(tx *bbolt.Tx) error) error
}

const (
	bucketName = "state"
	keyPrefix  = "edge."
)

var (
	ErrNotFound = errors.New("state: not found")
)

type Service struct {
	db    DB
	mx    *sync.RWMutex
	state objx.Map
}

func NewService(db DB, cleanStart bool) (Service, error) {
	err := db.Update(func(tx *bbolt.Tx) error {
		var err error

		if cleanStart {
			err = tx.DeleteBucket([]byte(bucketName))
			if err != nil && err != bbolt.ErrBucketNotFound {
				return err
			}
		}

		_, err = tx.CreateBucketIfNotExists([]byte(bucketName))

		return err
	})

	if err != nil {
		return Service{}, err
	}

	s := Service{db: db, mx: new(sync.RWMutex)}

	values, err := s.getAll()
	if err != nil {
		return Service{}, err
	}

	s.mx.Lock()
	s.state = convert(values)
	s.mx.Unlock()

	return s, nil
}

func (s Service) Get(key string) map[string]interface{} {
	s.mx.RLock()
	val := s.state.Get(key)
	s.mx.RUnlock()

	if val.IsNil() {
		return nil
	}

	vv := val.Data().([]byte)

	return Convert(key, vv)
}

func (s Service) Set(key string, v []byte) error {
	err := s.db.Update(func(tx *bbolt.Tx) error {
		bk := tx.Bucket([]byte(bucketName))
		return bk.Put([]byte(key), v)
	})
	if err != nil {
		return err
	}

	s.mx.Lock()
	s.state.Set(key, jsoniter.RawMessage(v))
	s.mx.Unlock()

	return nil
}

func Convert(key string, v []byte) map[string]interface{} {
	key = strings.TrimPrefix(key, keyPrefix)

	data := make(objx.Map, 1)

	data.Set(key, jsoniter.RawMessage(v))

	return data
}

func convert(state map[string][]byte) map[string]interface{} {
	data := make(objx.Map, len(state))

	for k, v := range state {
		k = strings.TrimPrefix(k, keyPrefix)
		data.Set(k, jsoniter.RawMessage(v))
	}

	return data
}

func (s Service) getAll() (map[string][]byte, error) {
	var values map[string][]byte

	err := s.db.View(func(tx *bbolt.Tx) error {
		bk := tx.Bucket([]byte(bucketName))

		values = make(map[string][]byte, bk.Stats().KeyN)

		return bk.ForEach(func(k, v []byte) error {
			values[string(k)] = v

			return nil
		})
	})

	return values, err
}
