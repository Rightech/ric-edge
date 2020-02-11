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
	s.state = values
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

	return map[string]interface{}{key: val.Data()}
}

func (s Service) Set(key string, v interface{}) error {
	s.mx.Lock()
	s.state.Set(key, v)
	s.mx.Unlock()

	data, err := jsoniter.ConfigFastest.Marshal(v)
	if err != nil {
		return err
	}

	err = s.db.Update(func(tx *bbolt.Tx) error {
		bk := tx.Bucket([]byte(bucketName))
		return bk.Put([]byte(key), data)
	})
	if err != nil {
		return err
	}

	return nil
}

func (s Service) getAll() (map[string]interface{}, error) {
	var values map[string]interface{}

	err := s.db.View(func(tx *bbolt.Tx) error {
		bk := tx.Bucket([]byte(bucketName))

		values = make(map[string]interface{}, bk.Stats().KeyN)

		return bk.ForEach(func(k, v []byte) error {
			var data interface{}

			err := jsoniter.ConfigFastest.Unmarshal(v, &data)
			if err != nil {
				return err
			}

			values[string(k)] = data

			return nil
		})
	})

	return values, err
}
