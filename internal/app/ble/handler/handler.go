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
	"encoding/base64"
	"errors"
	"fmt"
	"time"
	"unsafe"

	"github.com/Rightech/ric-edge/pkg/jsonrpc"
	"github.com/Rightech/ric-edge/third_party/go-ble/ble"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/objx"
)

func init() { // nolint: gochecknoinits
	// custom uuid decoder decode uuid as string
	// default decoder decode uuid as byte slice (base64)
	jsoniter.RegisterTypeEncoder("ble.UUID", bleUUIDDecoder{})
}

type Service struct {
	dev ble.Device
	rpc jsonrpc.RPC
	// this map address to client
	// it's required because only one client can exists
	// so when subscriptions starts we remember client and use it in next calls
	// when subscription dies client removes from map
	conns map[string]ble.Client
}

func (s *Service) InjectRPC(rpc jsonrpc.RPC) {
	// this method required for lazy initialization of rpc client
	// this client needs to send notifications (see subscribe)
	s.rpc = rpc
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
	case "ble-subscribe":
		res, err = s.subscribe(req.Params)
	case "ble-subscribe-cancel":
		res, err = s.subscribeCancel(req.Params)
	default:
		err = jsonrpc.ErrMethodNotFound.AddData("method", req.Method)
	}

	return
}

// get type of Eddystone beacon
func getBeaconType(beaconType int8) string {
	beacon := ""
	switch beaconType {
	case 0x00:
		beacon = "UID"
	case 0x10:
		beacon = "URL"
	case 0x20:
		beacon = "TLM"
	default:
		beacon = "undefined"
	}

	return beacon
}

type dev struct {
	Addr          string `json:"addr"`
	RSSI          int    `json:"rssi"`
	Name          string `json:"name"`
	Connectable   bool   `json:"connectable"`
	Eddystone     bool   `json:"eddystone"`
	EddystoneType string `json:"eddystone_type"`
	EddystoneURL  string `json:"eddystone_url"`
}

func (s Service) scan(params objx.Map) (interface{}, error) {

	prefixes := []string{"http://www.", "https://www.", "http://", "https://"}

	suffixes := []string{
		".com/",
		".org/",
		".edu/",
		".net/",
		".info/",
		".biz/",
		".gov/",
		".com",
		".org",
		".edu",
		".net",
		".info",
		".biz",
		".gov",
	}

	timeout, err := time.ParseDuration(params.Get("timeout").Str("5s"))

	if err != nil {
		return nil, jsonrpc.ErrInvalidParams.AddData("msg", err.Error())
	}

	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), timeout))

	devices := make(map[string]*dev)

	advHandler := func(a ble.Advertisement) {
		flag := false
		targetURL := ""
		beaconKind := ""

		len := len(a.Services())
		if len > 0 {
			service := a.Services()[0]
			if service.String() == "feaa" {
				for _, serviceData := range a.ServiceData() {

					eddystoneData := serviceData.Data
					beaconType := eddystoneData[:1]
					// txPower := eddystoneData[1:2]
					urlPrefix := eddystoneData[2:3]
					urlContent := string(eddystoneData[3 : cap(eddystoneData)-1])
					fmt.Print("\n", urlContent)
					urlSuffix := eddystoneData[cap(eddystoneData)-1]
					fmt.Print("\n", urlSuffix)

					prefix := prefixes[urlPrefix[0]]
					suffix := suffixes[urlSuffix]

					beaconKind = getBeaconType(int8(beaconType[0]))
					targetURL = prefix + urlContent + suffix

				}
				flag = true
			}
		}
		v, ok := devices[a.Addr().String()]
		if ok {
			v.Name = a.LocalName()
			v.RSSI = a.RSSI()
			v.Connectable = a.Connectable()
			v.Eddystone = flag
			v.EddystoneType = beaconKind
			v.EddystoneURL = targetURL

			return
		}

		devices[a.Addr().String()] = &dev{
			Addr:          a.Addr().String(),
			RSSI:          a.RSSI(),
			Name:          a.LocalName(),
			Connectable:   a.Connectable(),
			Eddystone:     flag,
			EddystoneURL:  targetURL,
			EddystoneType: beaconKind,
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
	address := params.Get("device").Str()
	if address == "" {
		return nil, jsonrpc.ErrInvalidParams.AddData("msg", "empty device")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var err error

	cli, ok := s.conns[address]
	if !ok {
		cli, err = s.dev.Dial(ctx, ble.NewAddr(address))
		if err != nil {
			return nil, err
		}
	}

	defer func() {
		if !ok {
			cli.CancelConnection() // nolint: errcheck
		}
	}()

	return cli.DiscoverProfile(true)
}

func parseRequest(params objx.Map) (a string, s ble.UUID, c ble.UUID, err error) {
	a = params.Get("device").Str()
	if a == "" {
		err = jsonrpc.ErrInvalidParams.AddData("msg", "empty device")
		return
	}

	s, err = ble.Parse(params.Get("service_uuid").Str())
	if err != nil {
		err = jsonrpc.ErrInvalidParams.AddData("p", "service_uuid").
			AddData("msg", err.Error())
		return
	}

	c, err = ble.Parse(params.Get("characteristic_uuid").Str())
	if err != nil {
		err = jsonrpc.ErrInvalidParams.AddData("p", "characteristic_uuid").
			AddData("msg", err.Error())
		return
	}

	return
}

func (s Service) read(params objx.Map) (interface{}, error) {
	address, srvUUID, chUUID, err := parseRequest(params)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cli, ok := s.conns[address]
	if !ok {
		cli, err = s.dev.Dial(ctx, ble.NewAddr(address))
		if err != nil {
			return nil, err
		}
	}

	defer func() {
		if !ok {
			cli.CancelConnection() // nolint: errcheck
		}
	}()

	srv, err := cli.DiscoverServices([]ble.UUID{srvUUID})
	if err != nil {
		return nil, err
	}

	ch, err := cli.DiscoverCharacteristics([]ble.UUID{chUUID}, srv[0])
	if err != nil {
		return nil, err
	}

	return cli.ReadCharacteristic(ch[0])
}

func (s Service) write(params objx.Map) (interface{}, error) {
	address, srvUUID, chUUID, err := parseRequest(params)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cli, ok := s.conns[address]
	if !ok {
		cli, err = s.dev.Dial(ctx, ble.NewAddr(address))
		if err != nil {
			return nil, err
		}
	}

	defer func() {
		if !ok {
			cli.CancelConnection() // nolint: errcheck
		}
	}()

	srv, err := cli.DiscoverServices([]ble.UUID{srvUUID})
	if err != nil {
		return nil, err
	}

	ch, err := cli.DiscoverCharacteristics([]ble.UUID{chUUID}, srv[0])
	if err != nil {
		return nil, err
	}

	vaL := params.Get("value").Str()

	value, err := base64.StdEncoding.DecodeString(vaL)
	if err != nil {
		value = []byte(vaL)
	}

	err = cli.WriteCharacteristic(ch[0], value, false)
	if err != nil {
		return nil, err
	}

	return true, nil
}

func (s Service) subscribe(params objx.Map) (interface{}, error) {
	address, srvUUID, chUUID, err := parseRequest(params)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cli, ok := s.conns[address]
	if !ok {
		cli, err = s.dev.Dial(ctx, ble.NewAddr(address))
		if err != nil {
			return nil, err
		}
	}

	srv, err := cli.DiscoverServices([]ble.UUID{srvUUID})
	if err != nil {
		return nil, err
	}

	ch, err := cli.DiscoverCharacteristics([]ble.UUID{chUUID}, srv[0])
	if err != nil {
		return nil, err
	}

	_, err = cli.DiscoverDescriptors(nil, ch[0])
	if err != nil {
		return nil, err
	}

	nfSrv := s.rpc.NewNotification(params)

	err = cli.Subscribe(ch[0], params.Get("indicator").Bool(), func(req []byte) {
		nfSrv.Send(req)
	})
	if err != nil {
		return nil, err
	}

	if !ok {
		s.conns[address] = cli
	}

	return nfSrv, nil
}

func (s Service) subscribeCancel(params objx.Map) (interface{}, error) {
	address := params.Get("device").Str()

	cli, ok := s.conns[address]
	if !ok {
		return nil, jsonrpc.ErrInvalidRequest.AddData("msg", "sub not found")
	}

	err := cli.ClearSubscriptions()
	if err != nil {
		return nil, err
	}

	err = cli.CancelConnection()
	if err != nil {
		return nil, err
	}

	delete(s.conns, address)

	return true, nil
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
