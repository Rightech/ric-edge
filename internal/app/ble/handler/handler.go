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
