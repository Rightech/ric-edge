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

package mqtt

import (
	"bytes"

	"github.com/eclipse/paho.mqtt.golang/packets"
	log "github.com/sirupsen/logrus"
	"go.etcd.io/bbolt"
)

type DB interface {
	Update(func(tx *bbolt.Tx) error) error
	View(func(tx *bbolt.Tx) error) error
}

type Service struct {
	db DB
}

const (
	bucketName = "mqtt"
)

func NewStore(db DB) Service {
	return Service{db}
}

func (s Service) Open() {
	err := s.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}

		for err = range tx.Check() {
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		log.WithError(err).Error("bolt.Open")
	}
}

func (s Service) Put(key string, message packets.ControlPacket) {
	err := s.db.Update(func(tx *bbolt.Tx) error {
		bk := tx.Bucket([]byte(bucketName))

		var buf []byte
		writer := bytes.NewBuffer(buf)

		err := message.Write(writer)
		if err != nil {
			return err
		}

		return bk.Put([]byte(key), writer.Bytes())
	})

	if err != nil {
		log.WithField("key", key).WithField("msg", message).WithError(err).
			Error("bolt.Put")
	}
}

func (s Service) Get(key string) packets.ControlPacket {
	var (
		packet packets.ControlPacket
		err    error
	)

	err = s.db.View(func(tx *bbolt.Tx) error {
		bk := tx.Bucket([]byte(bucketName))

		vl := bk.Get([]byte(key))

		packet, err = packets.ReadPacket(bytes.NewReader(vl))

		return err
	})

	if err != nil {
		log.WithField("key", key).WithError(err).Error("bolt.Get")
		return nil
	}

	return packet
}

func (s Service) All() []string {
	var keys []string

	err := s.db.View(func(tx *bbolt.Tx) error {
		bk := tx.Bucket([]byte(bucketName))

		keys = make([]string, 0, bk.Stats().KeyN)

		return bk.ForEach(func(k, _ []byte) error {
			keys = append(keys, string(k))
			return nil
		})
	})
	if err != nil {
		log.WithError(err).Error("bolt.All")
		return nil
	}

	return keys
}

func (s Service) Del(key string) {
	err := s.db.Update(func(tx *bbolt.Tx) error {
		bk := tx.Bucket([]byte(bucketName))
		return bk.Delete([]byte(key))
	})
	if err != nil {
		log.WithField("key", key).WithError(err).Error("bolt.Del")
	}
}

func (s Service) Close() {}

func (s Service) Reset() {
	err := s.db.Update(func(tx *bbolt.Tx) error {
		err := tx.DeleteBucket([]byte(bucketName))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucket([]byte(bucketName))
		return err
	})
	if err != nil {
		log.WithError(err).Error("bolt.Reset")
	}
}
