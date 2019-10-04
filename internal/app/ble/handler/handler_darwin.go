package handler

import "github.com/go-ble/ble/darwin"

func New() (Service, error) {
	dev, err := darwin.NewDevice()
	if err != nil {
		return Service{}, err
	}

	return Service{dev}, nil
}
