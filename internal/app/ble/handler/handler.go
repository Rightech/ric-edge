package handler

import (
	"errors"

	"github.com/Rightech/ric-edge/pkg/jsonrpc"
	"github.com/stretchr/objx"
)

type Service struct{}

func (s Service) Call(req jsonrpc.Request) (res interface{}, err error) {
	switch req.Method {
	case "ble-scan":
		res, err = s.scan(req.Params)
	case "ble-connect":
		res, err = s.connect(req.Params)
	case "ble-read":
		res, err = s.read(req.Params)
	case "ble-write":
		res, err = s.write(req.Params)
	default:
		err = jsonrpc.ErrMethodNotFound.AddData("method", req.Method)
	}
	return
}

func (s Service) scan(params objx.Map) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (s Service) connect(params objx.Map) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (s Service) read(params objx.Map) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (s Service) write(params objx.Map) (interface{}, error) {
	return nil, errors.New("not implemented")
}
