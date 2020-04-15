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
	"fmt"
	"strconv"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

// this rewrite of this example
// https://github.com/gopcua/opcua/blob/a2111b07cd2c925c83f0eabd292d8b2747645d11/examples/browse/browse.go

type nodeDef struct {
	NodeID      string             `json:"node_id"`
	NodeClass   ua.NodeClass       `json:"node_class"`
	AccessLevel ua.AccessLevelType `json:"access_level"`
	Writable    bool               `json:"writable"`
	BrowseName  string             `json:"browse_name"`
	Description string             `json:"description"`
	Path        string             `json:"path"`
	DataType    string             `json:"data_type"`
	Unit        string             `json:"unit"`
	Scale       string             `json:"scale"`
	Min         string             `json:"min"`
	Max         string             `json:"max"`
}

func (n nodeDef) records() []string {
	return []string{n.BrowseName, n.DataType, n.NodeID, n.Unit,
		n.Scale, n.Min, n.Max, strconv.FormatBool(n.Writable), n.Description}
}

func join(a, b string) string {
	if a == "" {
		return b
	}

	return a + "." + b
}

func browse(n *opcua.Node, path string, level int) ([]nodeDef, error) {
	if level > 10 {
		return nil, nil
	}

	attrs, err := n.Attributes(ua.AttributeIDNodeClass,
		ua.AttributeIDBrowseName, ua.AttributeIDDescription,
		ua.AttributeIDAccessLevel, ua.AttributeIDDataType)
	if err != nil {
		return nil, err
	}

	var def = nodeDef{
		NodeID: n.ID.String(),
	}

	err = fillStatus(attrs, &def)
	if err != nil {
		return nil, err
	}

	def.Path = join(path, def.BrowseName)

	return buildNodeList(def, n, level)
}

func fillStatus(attrs []*ua.DataValue, def *nodeDef) error {
	switch err := attrs[0].Status; err {
	case ua.StatusOK:
		def.NodeClass = ua.NodeClass(attrs[0].Value.Int())
	default:
		return err
	}

	switch err := attrs[1].Status; err {
	case ua.StatusOK:
		def.BrowseName = attrs[1].Value.String()
	default:
		return err
	}

	switch err := attrs[2].Status; err {
	case ua.StatusOK:
		def.Description = attrs[2].Value.String()
	case ua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return err
	}

	switch err := attrs[3].Status; err {
	case ua.StatusOK:
		def.AccessLevel = ua.AccessLevelType(attrs[3].Value.Uint())
		def.Writable = def.AccessLevel&ua.AccessLevelTypeCurrentWrite == ua.AccessLevelTypeCurrentWrite
	case ua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return err
	}

	return fillDataType(attrs, def)
}

func fillDataType(attrs []*ua.DataValue, def *nodeDef) error {
	switch err := attrs[4].Status; err {
	case ua.StatusOK:
		switch v := attrs[4].Value.NodeID().IntID(); v {
		case id.DateTime:
			def.DataType = "time.Time"
		case id.Boolean:
			def.DataType = "bool"
		case id.SByte:
			def.DataType = "int8"
		case id.Int16:
			def.DataType = "int16"
		case id.Int32:
			def.DataType = "int32"
		case id.Int64:
			def.DataType = "int64"
		case id.Byte:
			def.DataType = "byte"
		case id.UInt16:
			def.DataType = "uint16"
		case id.UInt32:
			def.DataType = "uint32"
		case id.UInt64:
			def.DataType = "uint64"
		case id.UtcTime:
			def.DataType = "time.Time"
		case id.String:
			def.DataType = "string"
		case id.Float:
			def.DataType = "float32"
		case id.Double:
			def.DataType = "float64"
		default:
			def.DataType = attrs[4].Value.NodeID().String()
		}
	case ua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return err
	}

	return nil
}

func buildNodeList(def nodeDef, n *opcua.Node, level int) ([]nodeDef, error) {
	var nodes []nodeDef

	if def.NodeClass == ua.NodeClassVariable {
		nodes = append(nodes, def)
	}

	browseChildren := func(refType uint32) error {
		refs, err := n.ReferencedNodes(refType, ua.BrowseDirectionForward,
			ua.NodeClassAll, true)
		if err != nil {
			return fmt.Errorf("references: %d: %w", refType, err)
		}

		for _, rn := range refs {
			children, err := browse(rn, def.Path, level+1)
			if err != nil {
				return fmt.Errorf("browse children: %w", err)
			}

			nodes = append(nodes, children...)
		}

		return nil
	}

	if err := browseChildren(id.HasComponent); err != nil {
		return nil, err
	}

	if err := browseChildren(id.Organizes); err != nil {
		return nil, err
	}

	if err := browseChildren(id.HasProperty); err != nil {
		return nil, err
	}

	return nodes, nil
}
