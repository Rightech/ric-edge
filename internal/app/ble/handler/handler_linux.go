package handler

import (
	"github.com/Rightech/ric-edge/third_party/go-ble/ble"
	"github.com/Rightech/ric-edge/third_party/go-ble/ble/linux"
)

func New() (Service, error) {
	dev, err := linux.NewDeviceWithName("ble-connector")
	if err != nil {
		return Service{}, err
	}

	return Service{dev: dev, conns: make(map[string]ble.Client)}, nil
}
