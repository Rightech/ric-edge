package main

import (
	"github.com/Rightech/ric-edge/third_party/go-ble/ble/linux"
)

func NewDevice() (*linux.Device, error) {
	return linux.NewDeviceWithName("ble-srv")
}
