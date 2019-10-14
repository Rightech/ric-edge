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

package cloud

import (
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/objx"
)

type Object struct {
	ID     string `json:"_id"`
	Models struct {
		ID string
	}
	Config objx.Map
}

type Children struct {
	ID       string
	Name     string
	Type     string
	DataType string
	Active   bool
	Edge     struct {
		Read struct {
			Type     string
			Command  string
			Interval string
		}
	}
	Command  string
	Params   map[string]interface{}
	Children []Children
}

type Model struct {
	ID   string `json:"_id"`
	Data struct {
		Children []Children
	}
	actions map[string]ActionConfig
}

func (m Model) Actions() map[string]ActionConfig {
	return m.actions
}

type command struct {
	Command string
	Params  map[string]interface{}
}

type ActionConfig struct {
	ID        string
	Connector string
	Type      string
	Interval  string
	Payload   string
}

func (m *Model) Prepare(o Object) {
	m.actions = make(map[string]ActionConfig)

	commands := make(map[string]Children)
	actionCommand := make(map[string]command)

	m.walk(nil, commands, actionCommand, m.Data.Children)
	m.afterWalk(commands, actionCommand, o)
}

type params struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      jsoniter.RawMessage    `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
}

func fillPayload(payload string, data map[string]interface{}) string {
	var pld params

	err := jsoniter.ConfigFastest.UnmarshalFromString(payload, &pld)
	if err != nil {
		panic(err)
	}

	for k, v := range pld.Params {
		if vv, ok := v.(string); ok && strings.HasPrefix(vv, "{{") &&
			strings.HasSuffix(vv, "}}") {
			val, ok := data[vv[2:len(vv)-2]]
			if !ok {
				continue
			}

			pld.Params[k] = val
		}
	}

	res, err := jsoniter.ConfigFastest.MarshalToString(pld)
	if err != nil {
		panic(err)
	}

	return res
}

func (m *Model) afterWalk(commands map[string]Children, acmd map[string]command, o Object) {
	for k, v := range m.actions {
		if vv, ok := commands[acmd[v.ID].Command]; ok {
			payload, ok := vv.Params["payload"].(string)
			if !ok {
				// remove action if payload not string
				delete(m.actions, k)
				continue
			}

			for k, val := range acmd[v.ID].Params {
				// fill params template from config
				if vv, ok := val.(string); ok && strings.HasPrefix(vv, "{{") &&
					strings.HasSuffix(vv, "}}") {
					keyPath := vv[2 : len(vv)-2]
					keyPath = strings.ReplaceAll(keyPath, "object.config.", "")

					acmd[v.ID].Params[k] = o.Config.Get(keyPath).Data()
				}
			}

			acmd[v.ID].Params["parent.id"] = v.ID
			v.Payload = fillPayload(payload, acmd[v.ID].Params)

			// update action in map
			m.actions[k] = v
			continue
		}

		// if command not found remove action
		delete(m.actions, k)
	}
}

func (m *Model) walk(path []string, commands map[string]Children,
	acmd map[string]command, children []Children) {
	for _, c := range children {
		if !c.Active {
			continue
		}

		// add id to path
		// this id required to get connector type
		if c.Type == "subsystem" {
			path = append(path, c.ID)
		}

		if c.Type == "action" {
			c.Children = nil
			commands[c.ID] = c
			continue
		}

		if c.Edge.Read.Command != "" {
			ac := ActionConfig{
				ID:        c.ID,
				Connector: path[len(path)-2],
				Type:      c.Edge.Read.Type,
				Interval:  c.Edge.Read.Interval,
			}

			for _, cc := range c.Children {
				if cc.ID == c.Edge.Read.Command {
					acmd[c.ID] = command{
						Command: cc.Command,
						Params:  cc.Params,
					}
				}
			}

			m.actions[c.Name] = ac

			continue
		}

		m.walk(path, commands, acmd, c.Children)
		n := len(path) - 1
		if n > -1 {
			path = path[:n]
		}
	}
}
