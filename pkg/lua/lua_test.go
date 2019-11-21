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

package lua

import (
	"encoding/binary"
	"reflect"
	"testing"

	jsoniter "github.com/json-iterator/go"
)

func TestLuaBinaryOk(t *testing.T) {
	fn := `
	return binary_to_num(param)
	`
	l := New()

	err := l.Add("test", fn)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	bt := make([]byte, 4)
	binary.LittleEndian.PutUint32(bt, 152)

	res, err := l.Execute("test", bt)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if res.(float64) != float64(152) {
		t.Error("wrong result", res)
	}
}

func TestLuaBinaryErr(t *testing.T) {
	fn := `
	return binary_to_num(param)
	`
	l := New()

	err := l.Add("test", fn)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	bt := make([]byte, 4)
	binary.LittleEndian.PutUint32(bt, 152)

	res, err := l.Execute("test", bt[:2])

	if res != nil {
		t.Error("result should be nil")
		t.FailNow()
	}

	if err == nil {
		t.Error("error should not be nil")
	}
}

func TestLuaJSON(t *testing.T) {
	fn := `
	return from_json(param)
	`

	l := New()

	err := l.Add("test", fn)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	param := map[string]interface{}{
		"test":  true,
		"value": float64(1),
	}

	data, err := jsoniter.ConfigFastest.Marshal(param)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	res, err := l.Execute("test", data)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	result := res.(map[string]interface{})

	if !reflect.DeepEqual(param, result) {
		t.Error("param != result", param, result)
	}
}
