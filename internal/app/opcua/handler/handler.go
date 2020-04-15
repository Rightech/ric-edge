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
	"fmt"
	"strconv"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/objx"

	"github.com/Rightech/ric-edge/pkg/jsonrpc"
)

type Service struct {
	cli *opcua.Client
}

func New(endpoint, encryption, mode, serverCert, serverKey string) (Service, error) {
	var opts []opcua.Option

	switch mode {
	case "None":
		opts = []opcua.Option{
			opcua.SecurityModeString(mode),
		}
	default:
		endpoints, err := opcua.GetEndpoints(endpoint)
		if err != nil {
			return Service{}, err
		}
		ep := opcua.SelectEndpoint(endpoints, encryption, ua.MessageSecurityModeFromString(mode))
		if ep == nil {
			return Service{}, errors.New("opcua.new: failed to find suitable endpoint")
		}
		opts = []opcua.Option{
			opcua.SecurityPolicy(encryption),
			opcua.SecurityModeString(mode),
			opcua.CertificateFile(serverCert),
			opcua.PrivateKeyFile(serverKey),
			opcua.AuthAnonymous(),
			opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
		}
	}

	c := opcua.NewClient(endpoint, opts...)

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
		//	case "opcua-subscribe":
		//		res, err = s.subscribe(req.Params)
	default:
		err = jsonrpc.ErrMethodNotFound.AddData("method", req.Method)
	}

	return
}

func (s Service) read(params objx.Map) (interface{}, error) {
	nodeID, err := ua.ParseNodeID(params.Get("node_id").Str())
	if err != nil {
		return nil, fmt.Errorf("invalid node id: %w", err)
	}

	req := &ua.ReadRequest{
		MaxAge:             2000,
		NodesToRead:        []*ua.ReadValueID{{NodeID: nodeID}},
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

func getType(value string, c *opcua.Client) (ua.TypeID, error) {
	nodeID, _ := ua.ParseNodeID(value)

	req := &ua.ReadRequest{
		MaxAge: 2000,
		NodesToRead: []*ua.ReadValueID{
			{NodeID: nodeID},
		},
		TimestampsToReturn: ua.TimestampsToReturnBoth,
	}

	resp, err := c.Read(req)
	if err != nil {
		return ua.TypeID(0), fmt.Errorf("read while write failed: %w", err)
	}

	if resp.Results[0].Status != ua.StatusOK {
		return ua.TypeID(0), fmt.Errorf("status not OK: %d", resp.Results[0].Status)
	}

	return resp.Results[0].Value.Type(), nil
}

func (s Service) write(params objx.Map) (*ua.StatusCode, error) {
	nodeID := params.Get("node_id").Str()

	ID, err := ua.ParseNodeID(nodeID)
	if err != nil {
		return nil, fmt.Errorf("invalid node id: %w", err)
	}

	nodeType, err := getType(nodeID, s.cli)
	if err != nil {
		return nil, err
	}

	switch nodeType {
	case id.Boolean:
		input, err := strconv.ParseBool(params.Get("value").Str())
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}

		v, err := ua.NewVariant(input)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}

		req := &ua.WriteRequest{
			NodesToWrite: []*ua.WriteValue{
				{
					NodeID:      ID,
					AttributeID: ua.AttributeIDValue,
					Value: &ua.DataValue{
						EncodingMask: ua.DataValueValue,
						Value:        v, // new value of node
					},
				},
			},
		}

		resp, err := s.cli.Write(req)
		if err != nil {
			return nil, fmt.Errorf("write failed: %w", err)
		}
		return &resp.Results[0], nil

	case id.Int32:
		input, err := strconv.ParseInt(params.Get("value").Str(), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}
		result := int32(input)
		v, err := ua.NewVariant(result)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}

		req := &ua.WriteRequest{
			NodesToWrite: []*ua.WriteValue{
				{
					NodeID:      ID,
					AttributeID: ua.AttributeIDValue,
					Value: &ua.DataValue{
						EncodingMask: ua.DataValueValue,
						Value:        v, // new value of node
					},
				},
			},
		}

		resp, err := s.cli.Write(req)
		if err != nil {
			return nil, fmt.Errorf("write failed: %w", err)
		}
		return &resp.Results[0], nil

	case id.UInt32:
		input, err := strconv.ParseUint(params.Get("value").Str(), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}
		result := uint32(input)
		v, err := ua.NewVariant(result)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}

		req := &ua.WriteRequest{
			NodesToWrite: []*ua.WriteValue{
				{
					NodeID:      ID,
					AttributeID: ua.AttributeIDValue,
					Value: &ua.DataValue{
						EncodingMask: ua.DataValueValue,
						Value:        v, // new value of node
					},
				},
			},
		}

		resp, err := s.cli.Write(req)
		if err != nil {
			return nil, fmt.Errorf("write failed: %w", err)
		}
		return &resp.Results[0], nil

	case id.Int64:
		input, err := strconv.ParseInt(params.Get("value").Str(), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}

		v, err := ua.NewVariant(input)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}

		req := &ua.WriteRequest{
			NodesToWrite: []*ua.WriteValue{
				{
					NodeID:      ID,
					AttributeID: ua.AttributeIDValue,
					Value: &ua.DataValue{
						EncodingMask: ua.DataValueValue,
						Value:        v, // new value of node
					},
				},
			},
		}

		resp, err := s.cli.Write(req)
		if err != nil {
			return nil, fmt.Errorf("write failed: %w", err)
		}
		return &resp.Results[0], nil

	case id.UInt64:
		input, err := strconv.ParseUint(params.Get("value").Str(), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}

		v, err := ua.NewVariant(input)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}

		req := &ua.WriteRequest{
			NodesToWrite: []*ua.WriteValue{
				{
					NodeID:      ID,
					AttributeID: ua.AttributeIDValue,
					Value: &ua.DataValue{
						EncodingMask: ua.DataValueValue,
						Value:        v, // new value of node
					},
				},
			},
		}

		resp, err := s.cli.Write(req)
		if err != nil {
			return nil, fmt.Errorf("write failed: %w", err)
		}
		return &resp.Results[0], nil

	case id.Float:
		input, err := strconv.ParseFloat(params.Get("value").Str(), 32)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}
		result := float32(input)
		v, err := ua.NewVariant(result)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}

		req := &ua.WriteRequest{
			NodesToWrite: []*ua.WriteValue{
				{
					NodeID:      ID,
					AttributeID: ua.AttributeIDValue,
					Value: &ua.DataValue{
						EncodingMask: ua.DataValueValue,
						Value:        v, // new value of node
					},
				},
			},
		}

		resp, err := s.cli.Write(req)
		if err != nil {
			return nil, fmt.Errorf("write failed: %w", err)
		}
		return &resp.Results[0], nil

	case id.Double:
		input, err := strconv.ParseFloat(params.Get("value").Str(), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}

		v, err := ua.NewVariant(input)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}

		req := &ua.WriteRequest{
			NodesToWrite: []*ua.WriteValue{
				{
					NodeID:      ID,
					AttributeID: ua.AttributeIDValue,
					Value: &ua.DataValue{
						EncodingMask: ua.DataValueValue,
						Value:        v, // new value of node
					},
				},
			},
		}

		resp, err := s.cli.Write(req)
		if err != nil {
			return nil, fmt.Errorf("write failed: %w", err)
		}
		return &resp.Results[0], nil

	case id.String:
		v, err := ua.NewVariant(params.Get("value").Str())
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}

		req := &ua.WriteRequest{
			NodesToWrite: []*ua.WriteValue{
				{
					NodeID:      ID,
					AttributeID: ua.AttributeIDValue,
					Value: &ua.DataValue{
						EncodingMask: ua.DataValueValue,
						Value:        v, // new value of node
					},
				},
			},
		}

		resp, err := s.cli.Write(req)
		if err != nil {
			return nil, fmt.Errorf("write failed: %w", err)
		}
		return &resp.Results[0], nil

	case id.DateTime:
		layout := "2006-01-02 15:04:05.999999999 +0000 GMT"
		t, err := time.Parse(layout, params.Get("value").Str())
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}
		v, err := ua.NewVariant(t)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}

		req := &ua.WriteRequest{
			NodesToWrite: []*ua.WriteValue{
				{
					NodeID:      ID,
					AttributeID: ua.AttributeIDValue,
					Value: &ua.DataValue{
						EncodingMask: ua.DataValueValue,
						Value:        v, // new value of node
					},
				},
			},
		}

		resp, err := s.cli.Write(req)
		if err != nil {
			return nil, fmt.Errorf("write failed: %w", err)
		}
		return &resp.Results[0], nil

	default:
		return nil, fmt.Errorf("write failed: unsupported type of node - %v", nodeType)
	}
}

func (s Service) browse(params objx.Map) (interface{}, error) {
	nodeID, err := ua.ParseNodeID(params.Get("node_id").Str())
	if err != nil {
		return nil, fmt.Errorf("invalid node id: %w", err)
	}

	nodeList, err := browse(s.cli.Node(nodeID), "", 0)
	if err != nil {
		return nil, err
	}

	return nodeList, nil
}

func (s Service) Close() error {
	return s.cli.Close()
}
