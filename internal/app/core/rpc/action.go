package rpc

import (
	"github.com/Rightech/ric-edge/pkg/jsonrpc"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/objx"
)

func processActionResp(id string, resp []byte, err error) []byte {
	if err != nil {
		data := map[string]interface{}{
			"err": err.Error(),
		}

		if resp != nil {
			data["response"] = jsoniter.RawMessage(resp)
		}

		return jsonrpc.BuildErrResp(id, jsonrpc.ErrInternal.SetData(data))
	}

	if resp != nil {
		return jsonrpc.BuildResp(id, resp)
	}

	return jsonrpc.BuildResp(id, true)
}

func (s Service) addAction(id string, data objx.Map) []byte {
	body, err := jsoniter.ConfigFastest.Marshal(data.Get("params").Data())
	if err != nil {
		return jsonrpc.BuildErrResp(
			id, jsonrpc.ErrParse.AddData("msg", err.Error()))
	}
	resp, err := s.action.Add(body)

	return processActionResp(id, resp, err)
}

func (s Service) delAction(id string, data objx.Map) []byte {
	name := data.Get("params.name").Str()
	if name == "" {
		return jsonrpc.BuildErrResp(
			id, jsonrpc.ErrInvalidParams.AddData("msg", "name not found or not string"))
	}

	resp, err := s.action.Delete(name)

	return processActionResp(id, resp, err)
}

func (s Service) callAction(id string, data objx.Map) []byte {
	name := data.Get("params.name").Str()
	if name == "" {
		return jsonrpc.BuildErrResp(
			id, jsonrpc.ErrInvalidParams.AddData("msg", "name not found or not string"))
	}

	body, err := jsoniter.ConfigFastest.Marshal(data.Get("params.body").Data())
	if err != nil {
		return jsonrpc.BuildErrResp(
			id, jsonrpc.ErrParse.AddData("msg", err.Error()))
	}

	resp, err := s.action.Call(name, body)

	return processActionResp(id, resp, err)
}
