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

package ws

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Rightech/ric-edge/pkg/jsonrpc"
	jsoniter "github.com/json-iterator/go"
)

// build and write error response from error and status code
func writeError(w http.ResponseWriter, err error, code int) error {
	tags := []string{
		"error_" + strings.ToLower(
			strings.ReplaceAll(http.StatusText(code), " ", "_"),
		),
	}

	data, err := jsoniter.ConfigFastest.Marshal(map[string]interface{}{
		"success": false,
		"code":    code,
		"message": err.Error(),
		"tags":    tags,
	})

	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	_, err = w.Write(data)
	if err != nil {
		err = fmt.Errorf("writeError:%w", err)
	}

	return err
}

var (
	errNotFound     = jsonrpc.ErrServer.AddData("msg", "connector not found").SetCode(-32000)
	errNotAvailable = jsonrpc.ErrServer.AddData("msg", "connector not available").SetCode(-32001)
)
