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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Rightech/ric-edge/pkg/jsonrpc"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/objx"
)

type Service struct {
	cli *opcua.Client
}

func New(endpoint string) (Service, error) {
	c := opcua.NewClient(endpoint)

	if err := c.Connect(context.Background()); err != nil {
		return Service{}, err
	}

	return Service{c}, nil
}

func (s Service) Call(req jsonrpc.Request) (res interface{}, err error) {
	switch req.Method {
	case "opcua-read":
		res, err = s.read(req.Params)
	case "opcua-write":
		res, err = s.write(req.Params)
	case "opcua-browse":
		res, err = s.browse(req.Params)
	default:
		err = jsonrpc.ErrMethodNotFound.AddData("method", req.Method)
	}

	return
}

func (s Service) read(params objx.Map) (interface{}, error) {
	nodeID := params.Get("node_id").Str()

	id, err := ua.ParseNodeID(nodeID)
	if err != nil {
		return nil, fmt.Errorf("invalid node id: %w", err)
	}

	req := &ua.ReadRequest{
		MaxAge:             2000,
		NodesToRead:        []*ua.ReadValueID{{NodeID: id}},
		TimestampsToReturn: ua.TimestampsToReturnBoth,
	}

	resp, err := s.cli.Read(req)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	if resp.Results[0].Status != ua.StatusOK {
		return nil, fmt.Errorf("status not OK: %d", resp.Results[0].Status)
	}

	return resp.Results[0].Value.Value(), nil
}

func parseValue(value *objx.Value) (interface{}, error) {
	if num, ok := value.Data().(json.Number); ok {
		if i64, err := num.Int64(); err == nil {
			return i64, nil
		}

		if f64, err := num.Float64(); err == nil {
			return f64, nil
		}
	}

	if value.IsInterSlice() {
		slice := value.InterSlice()

		switch v := slice[0].(type) {
		case json.Number:
			_, err := v.Int64()

			if err == nil {
				results := make([]int64, len(slice))

				for i, v := range value.InterSlice() {
					results[i], err = v.(json.Number).Int64()
					if err != nil {
						return nil, fmt.Errorf("parse to int64: %w", err)
					}
				}

				return results, nil
			}

			results := make([]float64, len(slice))

			for i, v := range value.InterSlice() {
				results[i], err = v.(json.Number).Float64()
				if err != nil {
					return nil, fmt.Errorf("parse to double: %w", err)
				}
			}

			return results, nil
		case string:
			var (
				results = make([]string, len(slice))
				ok      bool
			)

			for i, v := range value.InterSlice() {
				results[i], ok = v.(string)
				if !ok {
					return nil, errors.New("parse: different types in array")
				}
			}

			return results, nil
		default:
			return nil, errors.New("parse: unknown array values type")
		}
	}

	return value.Data(), nil
}

func (s Service) write(params objx.Map) (*ua.StatusCode, error) {
	nodeID := params.Get("node_id").Str()

	id, err := ua.ParseNodeID(nodeID)
	if err != nil {
		return nil, fmt.Errorf("invalid node id: %w", err)
	}

	val, err := parseValue(params.Get("value"))
	if err != nil {
		return nil, err
	}

	v, err := ua.NewVariant(val)
	if err != nil {
		return nil, fmt.Errorf("invalid value: %w", err)
	}

	req := &ua.WriteRequest{
		NodesToWrite: []*ua.WriteValue{
			{
				NodeID:      id,
				AttributeID: ua.AttributeIDValue,
				Value: &ua.DataValue{
					EncodingMask: ua.DataValueValue,
					Value:        v,
				},
			},
		},
	}

	resp, err := s.cli.Write(req)
	if err != nil {
		return nil, fmt.Errorf("write failed: %w", err)
	}

	return &resp.Results[0], nil
}

func (s Service) browse(params objx.Map) (interface{}, error) {
	nodeID := params.Get("node_id").Str()

	id, err := ua.ParseNodeID(nodeID)
	if err != nil {
		return nil, fmt.Errorf("invalid node id: %w", err)
	}

	nodeList, err := browse(s.cli.Node(id), "", 0)
	if err != nil {
		return nil, err
	}

	return nodeList, nil
}

func (s Service) Close() error {
	return s.cli.Close()
}
