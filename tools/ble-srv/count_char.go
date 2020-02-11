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

package main

import (
	"encoding/binary"
	"math"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/Rightech/ric-edge/third_party/go-ble/ble"
)

func nfHandler(typ string) ble.NotifyHandler {
	return ble.NotifyHandlerFunc(func(req ble.Request, n ble.Notifier) {
		cnt := uint32(0)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		log.Printf("count: %s subscribed\n", typ)
		for {
			select {
			case <-n.Context().Done():
				log.Printf("count: %s unsubscribed\n", typ)
				return
			case <-ticker.C:
				log.Printf("count: %s: %d", typ, cnt)

				bs := make([]byte, 4)
				binary.LittleEndian.PutUint32(bs, cnt)

				if _, err := n.Write(bs); err != nil {
					// Client disconnected prematurely before unsubscription.
					log.WithError(err).Errorf("count: Failed to %s", typ)
					return
				}
				cnt++
			}
		}
	})
}

// newCountChar ...
func newCountChar() *ble.Characteristic {
	var n float64

	c := ble.NewCharacteristic(ble.MustParse("00010000-0002-1000-8000-00805F9B34FB"))
	c.HandleRead(ble.ReadHandlerFunc(func(req ble.Request, rsp ble.ResponseWriter) {
		log.Printf("count: Read %f", n)

		bs := make([]byte, 8)
		binary.LittleEndian.PutUint64(bs, math.Float64bits(n))

		_, err := rsp.Write(bs)
		if err != nil {
			log.WithError(err).Error("read: write err")
			rsp.SetStatus(ble.ErrInvalidHandle)
			return
		}

		rsp.SetStatus(ble.ErrSuccess)
	}))

	c.HandleWrite(ble.WriteHandlerFunc(func(req ble.Request, rsp ble.ResponseWriter) {
		val := math.Float64frombits(binary.LittleEndian.Uint64(req.Data()))

		log.Printf("count: Write %f", val)

		n = val

		rsp.SetStatus(ble.ErrSuccess)
	}))

	c.HandleNotify(nfHandler("notify"))
	c.HandleIndicate(nfHandler("indicate"))

	return c
}
