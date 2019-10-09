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

package rpc

import (
	"time"

	"github.com/Rightech/ric-edge/internal/pkg/core/cloud"
	"github.com/Rightech/ric-edge/pkg/jsonrpc"
	"github.com/Rightech/ric-edge/pkg/nanoid"
	"github.com/Rightech/ric-edge/pkg/store/state"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/objx"
)

type stater interface {
	Get(string) ([]byte, error)
	Set(string, []byte) error
}

type rpcCli interface {
	Call(name, id string, p []byte) <-chan []byte
}

type api interface {
	LoadModel(string) (cloud.Model, error)
	LoadObject(string) (cloud.Object, error)
}

type Service struct {
	rpc        rpcCli
	api        api
	obj        cloud.Object
	model      cloud.Model
	timeout    time.Duration
	state      stater
	requestsCh <-chan []byte
}

func New(id string, tm time.Duration, db state.DB, cleanStart bool, r rpcCli,
	api api, requestsCh <-chan []byte) (Service, error) {
	st, err := state.NewService(db, cleanStart)
	if err != nil {
		return Service{}, err
	}

	object, err := api.LoadObject(id)
	if err != nil {
		return Service{}, err
	}

	model, err := api.LoadModel(object.Models.ID)
	if err != nil {
		return Service{}, err
	}

	s := Service{r, api, object, model, tm, st, requestsCh}

	go s.requestsListener()

	return s, nil
}

var (
	errTimeout   = jsonrpc.ErrServer.AddData("msg", "timeout")
	errUnmarshal = jsonrpc.ErrParse.AddData("msg", "json unmarshal error")
	errBadIDType = jsonrpc.ErrInternal.AddData("msg", "id should be string or null")
)

func (s Service) requestsListener() {
	for msg := range s.requestsCh {
		// TODO: add implementation of call handler
		log.Info(string(msg))
	}
}

func (s Service) Call(name string, payload []byte) []byte {
	var data objx.Map
	err := jsoniter.ConfigFastest.Unmarshal(payload, &data)
	if err != nil {
		return jsonrpc.BuildErrResp("", errUnmarshal.AddData("err", err.Error()))
	}

	changed := false

	id := data.Get("id")

	if !id.IsNil() && !id.IsStr() {
		return jsonrpc.BuildErrResp("", errBadIDType.AddData("current_id", id.Data()))
	}

	if id.IsNil() {
		data.Set("id", nanoid.New())
		changed = true
	}

	device := data.Get("params.device")
	if !device.IsNil() && device.IsStr() {
		data.Set("params.device", s.obj.Config.Devs.Devs.Get(device.Str()))
		changed = true
	}

	if changed {
		payload, err = jsoniter.ConfigFastest.Marshal(data)
		if err != nil {
			panic(err)
		}
	}

	resultC := s.rpc.Call(name, data.Get("id").Str(), payload)
	timer := time.NewTimer(s.timeout)
	select {
	case msg := <-resultC:
		if !timer.Stop() {
			<-timer.C
		}
		s.updateState(data.Get("method").Str(), msg)
		return msg
	case <-timer.C:
		return jsonrpc.BuildErrResp(data.Get("id").Str(), errTimeout)
	}
}

func (s Service) updateState(method string, resp []byte) {
	result := struct {
		Result jsoniter.RawMessage
	}{}

	err := jsoniter.ConfigFastest.Unmarshal(resp, &result)
	if err != nil {
		log.WithFields(log.Fields{
			"value":  string(resp),
			"method": method,
			"error":  err,
		}).Error("updateState: unmarshal json")
		return
	}

	if len(result.Result) == 0 {
		return
	}

	err = s.state.Set(method, result.Result)
	if err != nil {
		log.WithFields(log.Fields{
			"value":  string(resp),
			"method": method,
			"error":  err,
		}).Error("updateState: set")
	}
}
