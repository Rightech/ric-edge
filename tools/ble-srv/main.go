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
	"context"
	"errors"

	log "github.com/sirupsen/logrus"

	"github.com/Rightech/ric-edge/internal/pkg/config"
	"github.com/Rightech/ric-edge/third_party/go-ble/ble"
)

func main() {
	config.Init(nil)

	d, err := newDevice()
	if err != nil {
		log.Fatalf("can't new device : %s", err)
	}

	ble.SetDefaultDevice(d)

	log.Info("Address: ", d.Address().String())

	testSvc := ble.NewService(ble.MustParse("00010000-0001-1000-8000-00805F9B34FB"))
	testSvc.AddCharacteristic(newCountChar())

	if err := ble.AddService(testSvc); err != nil {
		log.Fatalf("can't add service: %s", err)
	}

	ctx := ble.WithSigHandler(context.WithCancel(context.Background()))

	err = ble.AdvertiseNameAndServices(ctx, "ble-srv", testSvc.UUID)
	if errors.Is(err, context.Canceled) {
		log.Info("canceled")
	} else {
		log.Error(err)
	}
}
