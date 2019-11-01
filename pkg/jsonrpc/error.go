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
	"fmt"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/objx"
)

func init() { // nolint: gochecknoinits
	jsoniter.RegisterTypeEncoder("jsonrpc.Error", rpcErrEncoder{})
}

var (
	ErrParse          = Error{code: -32700, message: "Parse error"}
	ErrInvalidRequest = Error{code: -32600, message: "Invalid Request"}
	ErrMethodNotFound = Error{code: -32601, message: "Method not found"}
	ErrInvalidParams  = Error{code: -32602, message: "Invalid params"}
	ErrInternal       = Error{code: -32603, message: "Internal error"}
	ErrServer         = Error{code: -32000, message: "Server error"}
)

type Error struct {
	code    int
	message string
	data    objx.Map
}

// SetCode works only for ErrServer (code should be between -32099 and -32000)
func (e Error) SetCode(c int) Error {
	if -32099 <= e.code && e.code <= -32000 && -32099 <= c && c <= -32000 {
		e.code = c
	}

	return e
}

func (e Error) AddData(key string, value interface{}) Error {
	cp := e.data.Copy()
	cp[key] = value
	e.data = cp

	return e
}

func (e Error) SetData(data map[string]interface{}) Error {
	e.data = data
	return e
}

func (e Error) Error() string {
	return fmt.Sprintf("%d - %s", e.code, e.message)
}

// this is custom encoder for RPCError (we need it to encode private fields)
type rpcErrEncoder struct{}

func (rpcErrEncoder) IsEmpty(ptr unsafe.Pointer) bool {
	v := *((*Error)(ptr))
	return v.code == 0
}

func (rpcErrEncoder) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	v := *((*Error)(ptr))

	stream.WriteObjectStart()

	stream.WriteObjectField("code")
	stream.WriteInt(v.code)
	stream.WriteMore()

	stream.WriteObjectField("message")
	stream.WriteString(v.message)

	if len(v.data) != 0 {
		stream.WriteMore()
		stream.WriteObjectField("data")
		stream.WriteVal(v.data)
	}

	stream.WriteObjectEnd()
	stream.Flush()
}

func BuildErrResp(id string, e Error) []byte {
	data, err := jsoniter.ConfigFastest.MarshalToString(e)
	if err != nil {
		panic(err)
	}

	if id == "" {
		id = "null"
	}

	resp := `{"jsonrpc":"` + jsonRPCVersion + `","id":"` + id + `","error":` + data + `}`

	return []byte(resp)
}
