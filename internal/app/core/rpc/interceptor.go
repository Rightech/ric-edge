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
	"errors"
	"fmt"
	"strings"
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
	GetAll() (map[string][]byte, error)
}

type rpcCli interface {
	Call(name, id string, p []byte) <-chan []byte
}

type api interface {
	LoadModel(string) (cloud.Model, error)
	LoadObject(string) (cloud.Object, error)
}

type jober interface {
	AddFunc(string, func()) error
}

type Service struct {
	rpc        rpcCli
	api        api
	job        jober
	obj        cloud.Object
	model      cloud.Model
	timeout    time.Duration
	state      stater
	requestsCh <-chan []byte
}

func New(id string, tm time.Duration, db state.DB, cleanStart bool, r rpcCli,
	api api, j jober, requestsCh <-chan []byte) (Service, error) {
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

	s := Service{r, api, j, object, model, tm, st, requestsCh}

	go s.requestsListener()

	return s, s.spawnJobs(model.Actions())
}

var (
	errTimeout   = jsonrpc.ErrServer.AddData("msg", "timeout")
	errUnmarshal = jsonrpc.ErrParse.AddData("msg", "json unmarshal error")
	errBadIDType = jsonrpc.ErrInternal.AddData("msg", "id should be string or null")
)

func (s Service) requestsListener() {
	for msg := range s.requestsCh {
		request := struct {
			Params struct {
				RequestParams objx.Map `json:"__request_params"`
				Value         jsoniter.RawMessage
			}
		}{}

		err := jsoniter.ConfigFastest.Unmarshal(msg, &request)
		if err != nil {
			log.WithFields(log.Fields{
				"value": string(msg),
				"error": err,
			}).Error("updateState: unmarshal json")
			continue
		}

		if len(request.Params.Value) == 0 {
			log.WithField("value", string(msg)).
				Error("empty value in notification")
			continue
		}

		parent := request.Params.RequestParams.Get("_parent").Str()
		if parent == "" {
			log.WithField("value", string(msg)).Error("empty _parent in request")
			continue
		}

		log.WithFields(log.Fields{
			"parent": parent,
			"value":  string(request.Params.Value),
		}).Debug("new notification")

		err = s.state.Set(parent, request.Params.Value)
		if err != nil {
			log.WithFields(log.Fields{
				"value": string(msg),
				"param": parent,
				"error": err,
			}).Error("request set state: set")
			continue
		}
	}
}

func (s Service) buildJobFn(v cloud.ActionConfig) func() {
	return func() {
		resp := s.Call(v.Connector, v.Payload)
		log.WithField("r", string(resp)).Debug("cron job response")
	}
}

func (s Service) subscribe(v cloud.ActionConfig) {
	resp := s.Call(v.Connector, v.Payload)
	idVal := jsoniter.ConfigFastest.Get(resp, "result").Get("process_id")
	if idVal.LastError() != nil {
		log.WithError(idVal.LastError()).Error("process_id not found")
		return
	}

	log.Debug("start subscribe with process_id: ", idVal.ToString())
}

func (s Service) spawnJobs(actions map[string]cloud.ActionConfig) error {
	for _, v := range actions {
		switch v.Type {
		case "schedule":
			err := s.job.AddFunc(v.Interval, s.buildJobFn(v))
			if err != nil {
				return fmt.Errorf("spawn [%s]: %w", v.ID, err)
			}
		case "subscribe":
			s.subscribe(v)
		default:
			return errors.New("spawn: wrong type " + v.Type)
		}
	}

	return nil
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
	if !device.IsNil() && device.IsStr() && strings.HasPrefix(device.Str(), "{{") {
		devID := device.Str()
		devID = devID[2 : len(devID)-2]
		devID = strings.ReplaceAll(devID, "object.config.", "")
		data.Set("params.device", s.obj.Config.Get(devID).Str())
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
		s.updateState(data, msg)
		return msg
	case <-timer.C:
		return jsonrpc.BuildErrResp(data.Get("id").Str(), errTimeout)
	}
}

func (s Service) updateState(req objx.Map, resp []byte) {
	parent := req.Get("params._parent").Str()
	if parent == "" {
		return
	}

	result := struct {
		Result jsoniter.RawMessage
	}{}

	err := jsoniter.ConfigFastest.Unmarshal(resp, &result)
	if err != nil {
		log.WithFields(log.Fields{
			"value":  string(resp),
			"method": req.Get("method").Str(),
			"error":  err,
		}).Error("updateState: unmarshal json")
		return
	}

	if len(result.Result) == 0 {
		return
	}

	err = s.state.Set(parent, result.Result)
	if err != nil {
		log.WithFields(log.Fields{
			"value":  string(resp),
			"parent": parent,
			"method": req.Get("method").Str(),
			"error":  err,
		}).Error("updateState: set")
	}
}
