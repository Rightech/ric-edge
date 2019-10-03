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

package jsonrpc

import (
	"context"
	"errors"
	"io"
	"runtime/debug"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/objx"
)

// Caller interface
// this method invokes on request
// and encode response as jsonrpc response
type Caller interface {
	Call(Request) (interface{}, error)
}

const (
	jsonRPCVersion = "2.0"
)

type NextWriter interface {
	NextWriter() (io.WriteCloser, error)
}

type NextReader interface {
	NextReader() (io.Reader, error)
}

type Transport interface {
	NextWriter
	NextReader
}

type Service struct {
	tr Transport
	c  Caller
}

func New(tr Transport, c Caller) Service {
	return Service{tr, c}
}

// return decoder with UseNumber enabled
func newDecoder(r NextReader) (*jsoniter.Decoder, error) {
	reader, err := r.NextReader()
	if err != nil {
		return nil, err
	}

	decoder := jsoniter.ConfigFastest.NewDecoder(reader)
	decoder.UseNumber()
	return decoder, nil
}

func newEncoder(w NextWriter) (*jsoniter.Stream, io.Closer, error) {
	writer, err := w.NextWriter()
	if err != nil {
		return nil, nil, err
	}

	// we can't use jsoniter.Encoder because on Encode it writes \n at the end
	// so we use pure stream
	stream := jsoniter.ConfigFastest.BorrowStream(writer)
	return stream, writer, nil
}

func (s Service) Serve(ctx context.Context) error {
	for ctx.Err() == nil {
		decoder, err := newDecoder(s.tr) // json decoder
		if err != nil {
			return err
		}

		encoder, closer, err := newEncoder(s.tr) // json encoder
		if err != nil {
			return err
		}

		var req Request
		err = decoder.Decode(&req)
		if errors.Is(err, io.EOF) {
			// if error is a EOF we should return error
			// in this case service will try reconnect to transport
			// if it run via ServeWithReconnect
			// otherwise we should return jsonrpc response with error
			return err
		}

		res := s.handleMessage(req, err)

		encoder.WriteVal(res)
		encoder.Flush()
		closer.Close()

		if encoder.Error != nil {
			panic(encoder.Error)
		}
	}

	return nil
}

type Request struct {
	JSONRPC string
	Method  string
	ID      jsoniter.RawMessage
	Params  objx.Map
}

type response struct {
	JSONRPC string              `json:"jsonrpc"`
	ID      jsoniter.RawMessage `json:"id"`
	Result  interface{}         `json:"result,omitempty"`
	Error   *Error              `json:"error,omitempty"`
}

func (s Service) call(req Request) (res interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			res = nil
			err = ErrServer.SetData(map[string]interface{}{
				"msg":   r,
				"panic": true,
				"stack": string(debug.Stack()),
			}).SetCode(-32099)
		}
	}()

	res, err = s.c.Call(req)
	return
}

var (
	errBadVer    = ErrInvalidRequest.AddData("msg", "bad jsonrpc version")
	errBadMethod = ErrInvalidRequest.AddData("msg", "empty method")
)

func (s Service) handleMessage(req Request, err error) response {
	if err != nil {
		return buildResult(nil, nil, ErrParse.AddData("msg", err.Error()))
	}

	if req.JSONRPC != jsonRPCVersion {
		return buildResult(req.ID, nil, errBadVer)
	}

	if req.Method == "" {
		return buildResult(req.ID, nil, errBadMethod)
	}

	res, err := s.call(req)

	if res == nil && err == nil {
		panic("result and error are nil")
	}

	return buildResult(req.ID, res, err)
}

func buildResult(id jsoniter.RawMessage, res interface{}, e error) response {
	resp := response{
		ID:      id,
		JSONRPC: jsonRPCVersion,
	}

	if e != nil {
		res = nil
		rerr, ok := e.(Error)
		if !ok {
			rerr = ErrServer.AddData("msg", e.Error()).SetCode(-32098)
		}
		resp.Error = &rerr
	}

	resp.Result = res

	if len(resp.ID) == 0 {
		resp.ID = []byte("null")
	}

	return resp
}
