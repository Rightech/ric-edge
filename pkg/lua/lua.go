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
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
)

type Service struct {
	funcs map[string]*lua.FunctionProto
	base  *lua.LState
}

func New() Service {
	return Service{
		make(map[string]*lua.FunctionProto),
		newState(),
	}
}
func (s Service) Add(name, code string) error {
	fn, err := compile(name, code)
	if err != nil {
		return err
	}

	s.funcs[name] = fn

	return nil
}

func (s Service) Remove(name string) {
	delete(s.funcs, name)
}

func (s Service) Execute(name string, data interface{}) (interface{}, error) {
	fnProto, ok := s.funcs[name]
	if !ok {
		return nil, errors.New("function not found")
	}

	param := toVal(data)
	if param == nil {
		return nil, errors.New("unknown param type")
	}

	ls, _ := s.base.NewThread()
	defer ls.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	ls.SetContext(ctx)

	fn := ls.NewFunctionFromProto(fnProto)

	ls.SetGlobal("param", param)
	ls.Push(fn)

	err := ls.PCall(0, 1, nil)
	if err != nil {
		return nil, err
	}

	if ls.GetTop() != 1 {
		return nil, errors.New("wrong number of return values, should be 1")
	}

	return valTo(ls.Get(-1)), nil
}

func newState() *lua.LState {
	ls := lua.NewState(lua.Options{
		SkipOpenLibs: true,
	})

	lua.OpenPackage(ls)
	lua.OpenBase(ls)
	lua.OpenTable(ls)
	lua.OpenMath(ls)
	lua.OpenString(ls)

	ls.SetGlobal("binary_to_num", ls.NewFunction(binaryToNumber))
	ls.SetGlobal("num_to_binary", ls.NewFunction(numberToBinary))
	ls.SetGlobal("from_json", ls.NewFunction(fromJSON))

	return ls
}

func compile(name, code string) (*lua.FunctionProto, error) {
	chunk, err := parse.Parse(strings.NewReader(code), name)
	if err != nil {
		return nil, err
	}

	proto, err := lua.Compile(chunk, name)
	if err != nil {
		return nil, err
	}

	return proto, nil
}

func toVal(value interface{}) lua.LValue {
	if value == nil {
		return lua.LNil
	}

	if lval, ok := value.(lua.LValue); ok {
		return lval
	}

	if val, ok := value.(jsoniter.RawMessage); ok {
		return lua.LString(string(val))
	}

	if val, ok := value.([]byte); ok {
		return lua.LString(string(val))
	}

	switch val := reflect.ValueOf(value); val.Kind() {
	case reflect.String:
		return lua.LString(val.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return lua.LNumber(float64(val.Int()))
	case reflect.Bool:
		return lua.LBool(val.Bool())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return lua.LNumber(float64(val.Uint()))
	case reflect.Float32, reflect.Float64:
		return lua.LNumber(val.Float())
	default:
		return nil
	}
}

func valTo(value lua.LValue) interface{} {
	switch v := value.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(v)
	case lua.LString:
		return string(v)
	case lua.LNumber:
		return float64(v)
	case *lua.LTable:
		maxn := v.MaxN()
		if maxn == 0 { // table
			ret := make(map[string]interface{})

			v.ForEach(func(key, value lua.LValue) {
				ret[fmt.Sprint(valTo(key))] = valTo(value)
			})

			return ret
		}
		// array
		ret := make([]interface{}, 0, maxn)

		for i := 1; i <= maxn; i++ {
			ret = append(ret, valTo(v.RawGetInt(i)))
		}

		return ret
	default:
		return v
	}
}
