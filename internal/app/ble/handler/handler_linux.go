package handler

import (
	"github.com/go-ble/ble/linux"
)

func New() (Service, error) {
	dev, err := linux.NewDeviceWithName("ble-connector")
	if err != nil {
		return Service{}, err
	}

	return Service{dev}, nil
}
