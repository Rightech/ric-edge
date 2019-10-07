package main

import (
	"context"
	"errors"

	"github.com/Rightech/ric-edge/internal/pkg/config"
	"github.com/Rightech/ric-edge/third_party/go-ble/ble"
	log "github.com/sirupsen/logrus"
)

func main() {
	config.Init(nil)

	d, err := NewDevice()
	if err != nil {
		log.Fatalf("can't new device : %s", err)
	}
	ble.SetDefaultDevice(d)

	log.Info("Address: ", d.Address().String())

	testSvc := ble.NewService(ble.MustParse("00010000-0001-1000-8000-00805F9B34FB"))
	testSvc.AddCharacteristic(NewCountChar())

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
