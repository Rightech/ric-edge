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

package handler

import (
	"context"
	"errors"
	"time"
	"unsafe"

	"github.com/Rightech/ric-edge/pkg/jsonrpc"
	"github.com/Rightech/ric-edge/third_party/go-ble/ble"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/objx"
)

func init() { // nolint: gochecknoinits
	jsoniter.RegisterTypeEncoder("ble.UUID", bleUUIDDecoder{})
}

type Service struct {
	dev ble.Device
}

func (s Service) Call(req jsonrpc.Request) (res interface{}, err error) {
	switch req.Method {
	case "ble-scan":
		res, err = s.scan(req.Params)
	case "ble-discover":
		res, err = s.discover(req.Params)
	case "ble-read":
		res, err = s.read(req.Params)
	case "ble-write":
		res, err = s.write(req.Params)
	default:
		err = jsonrpc.ErrMethodNotFound.AddData("method", req.Method)
	}
	return
}

type dev struct {
	Addr        string `json:"addr"`
	RSSI        int    `json:"rssi"`
	Name        string `json:"name"`
	Connectable bool   `json:"connectable"`
}

func (s Service) scan(params objx.Map) (interface{}, error) {
	timeout, err := time.ParseDuration(params.Get("timeout").Str("5s"))
	if err != nil {
		return nil, jsonrpc.ErrInvalidParams.AddData("msg", err.Error())
	}

	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), timeout))

	devices := make(map[string]*dev)

	advHandler := func(a ble.Advertisement) {
		v, ok := devices[a.Addr().String()]
		if ok {
			v.Name = a.LocalName()
			v.RSSI = a.RSSI()
			v.Connectable = a.Connectable()
			return
		}

		devices[a.Addr().String()] = &dev{
			Addr:        a.Addr().String(),
			RSSI:        a.RSSI(),
			Name:        a.LocalName(),
			Connectable: a.Connectable(),
		}
	}

	err = s.dev.Scan(ctx, false, advHandler)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	}

	return mapToList(devices), nil
}

func mapToList(mp map[string]*dev) []dev {
	lst := make([]dev, 0, len(mp))

	for _, v := range mp {
		lst = append(lst, *v)
	}

	return lst
}

func (s Service) discover(params objx.Map) (interface{}, error) {
	address := params.Get("address").Str()
	if address == "" {
		return nil, jsonrpc.ErrInvalidParams.AddData("msg", "empty address")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cli, err := s.dev.Dial(ctx, ble.NewAddr(address))
	if err != nil {
		return nil, err
	}

	defer cli.CancelConnection() // nolint: errcheck

	return cli.DiscoverProfile(true)
}

func (s Service) read(params objx.Map) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (s Service) write(params objx.Map) (interface{}, error) {
	return nil, errors.New("not implemented")
}

type bleUUIDDecoder struct{}

func (bleUUIDDecoder) IsEmpty(ptr unsafe.Pointer) bool {
	v := *((*ble.UUID)(ptr))
	return v.Len() == 0
}

func (bleUUIDDecoder) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	v := *((*ble.UUID)(ptr))
	stream.WriteString(v.String())
}
