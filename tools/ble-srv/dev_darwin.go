package main

import "github.com/Rightech/ric-edge/third_party/go-ble/ble/darwin"

func NewDevice() (*darwin.Device, error) {
	return darwin.NewDevice()
}
