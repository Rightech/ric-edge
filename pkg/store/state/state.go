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
	"fmt"

	"github.com/etcd-io/bbolt"
)

type DB interface {
	Update(func(tx *bbolt.Tx) error) error
	View(func(tx *bbolt.Tx) error) error
}

const (
	bucketName = "state"
)

var (
	ErrNotFound = errors.New("state: not found")
)

type Service struct {
	db DB
}

func NewService(db DB, cleanStart bool) (Service, error) {
	err := db.Update(func(tx *bbolt.Tx) error {
		var err error

		if cleanStart {
			err = tx.DeleteBucket([]byte(bucketName))
			if err != nil {
				return err
			}
		}

		_, err = tx.CreateBucketIfNotExists([]byte(bucketName))

		return err
	})

	return Service{db}, err
}

func (s Service) Get(key string) ([]byte, error) {
	var data []byte

	err := s.db.View(func(tx *bbolt.Tx) error {
		bk := tx.Bucket([]byte(bucketName))
		data = bk.Get([]byte(key))
		return nil
	})

	if len(data) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, key)
	}

	return data, err
}

func (s Service) Set(key string, v []byte) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bk := tx.Bucket([]byte(bucketName))
		return bk.Put([]byte(key), v)
	})
}

func (s Service) GetAll() (map[string][]byte, error) {
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
