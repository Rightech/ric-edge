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
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	g "github.com/soniah/gosnmp"
	"github.com/stretchr/objx"

	"github.com/Rightech/ric-edge/pkg/jsonrpc"
	"github.com/Rightech/ric-edge/pkg/log/logger"
)

type Service struct {
	cli *g.GoSNMP
}

func versionToVersion(version string) (g.SnmpVersion, error) {
	switch version {
	case "1":
		return g.Version1, nil
	case "2c":
		return g.Version2c, nil
	case "3":
		return g.Version3, nil
	default:
		return g.SnmpVersion(^uint8(0)), errors.New("snmp: unknown version")
	}
}

func New(hostPort, community, version string) (Service, error) {
	if hostPort == "" {
		return Service{}, errors.New("snmp.new: empty host_port")
	}

	if community == "" {
		return Service{}, errors.New("snmp.new: empty community")
	}

	ver, err := versionToVersion(version)
	if err != nil {
		return Service{}, err
	}

	tmp := strings.Split(hostPort, ":")
	target := tmp[0]
	portS := tmp[1]

	port, err := strconv.ParseUint(portS, 10, 16)
	if err != nil {
		return Service{}, err
	}

	cli := &g.GoSNMP{
		Target:             target,
		Port:               uint16(port),
		Transport:          "udp",
		Community:          community,
		Version:            ver,
		Timeout:            time.Duration(2) * time.Second,
		Retries:            3,
		ExponentialTimeout: true,
		MaxOids:            g.MaxOids,
		Logger:             logger.New("info", log.DebugLevel),
	}

	err = cli.Connect()
	if err != nil {
		return Service{}, err
	}

	return Service{cli}, nil
}

func (s Service) Call(req jsonrpc.Request) (res interface{}, err error) {
	switch req.Method {
	case "snmp-get":
		res, err = s.get(req.Params)
	case "snmp-get-next":
		res, err = s.getNext(req.Params)
	case "snmp-get-bulk":
		res, err = s.getBulk(req.Params)
	case "snmp-walk":
		res, err = s.walk(req.Params)
	case "snmp-bulk-walk":
		res, err = s.bulkWalk(req.Params)
	case "snmp-set":
		res, err = s.set(req.Params)
	case "snmp-send-trap":
		res, err = s.sendTrap(req.Params)
	default:
		err = jsonrpc.ErrMethodNotFound.AddData("method", req.Method)
	}

	return
}

var (
	errBadOid = jsonrpc.ErrInvalidParams.AddData(
		"msg", "oid required and should be string array")
)

func parseOids(params objx.Map) ([]string, error) {
	val := params.Get("oids")

	if !val.IsInterSlice() {
		return nil, errBadOid
	}

	oids := make([]string, len(val.InterSlice()))

	for i, v := range val.InterSlice() {
		vv, ok := v.(string)
		if !ok {
			return nil, errBadOid
		}

		oids[i] = vv
	}

	return oids, nil
}

func encodeDataUnit(p g.SnmpPDU) map[string]interface{} {
	return map[string]interface{}{
		"oid":      p.Name,
		"type_num": p.Type,
		"type_str": p.Type.String(),
		"value":    p.Value,
	}
}

func encodeSnmpPacket(p *g.SnmpPacket) interface{} {
	result := make([]map[string]interface{}, len(p.Variables))
	for i, v := range p.Variables {
		result[i] = encodeDataUnit(v)
	}

	return result
}

const (
	maxUint8 = int64(^uint8(0))
	minUint8 = int64(0)
)

func getUint8(params objx.Map, k string) (uint8, error) {
	number, ok := params.Get(k).Data().(json.Number)
	if !ok {
		return 0, jsonrpc.ErrInvalidParams.AddData("msg", k+" required and should be number")
	}

	var (
		value int64
		err   error
	)

	if value, err = number.Int64(); err != nil {
		return 0, jsonrpc.ErrInvalidParams.AddData("msg", k+" required and should be number")
	}

	if !(minUint8 <= value && value <= maxUint8) {
		return 0, jsonrpc.ErrInvalidParams.AddData("msg", k+" should be uint8")
	}

	return uint8(value), nil
}

func (s Service) get(params objx.Map) (interface{}, error) {
	oids, err := parseOids(params)
	if err != nil {
		return nil, err
	}

	res, err := s.cli.Get(oids)
	if err != nil {
		return nil, err
	}

	return encodeSnmpPacket(res), nil
}

func (s Service) getNext(params objx.Map) (interface{}, error) {
	oids, err := parseOids(params)
	if err != nil {
		return nil, err
	}

	res, err := s.cli.GetNext(oids)
	if err != nil {
		return nil, err
	}

	return encodeSnmpPacket(res), nil
}

func (s Service) getBulk(params objx.Map) (interface{}, error) {
	oids, err := parseOids(params)
	if err != nil {
		return nil, err
	}

	nonRep, err := getUint8(params, "non_repeaters")
	if err != nil {
		return nil, err
	}

	maxRep, err := getUint8(params, "max_repetitions")
	if err != nil {
		return nil, err
	}

	res, err := s.cli.GetBulk(oids, nonRep, maxRep)
	if err != nil {
		return nil, err
	}

	return encodeSnmpPacket(res), nil
}

func (s Service) walk(params objx.Map) (interface{}, error) {
	oidV := params.Get("oid")
	if !oidV.IsStr() {
		return nil, jsonrpc.ErrInvalidParams.AddData("msg", "oid required and should be string")
	}

	result := make([]map[string]interface{}, 0)

	err := s.cli.Walk(oidV.Str(), func(dataUnit g.SnmpPDU) error {
		result = append(result, encodeDataUnit(dataUnit))
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s Service) bulkWalk(params objx.Map) (interface{}, error) {
	oidV := params.Get("oid")
	if !oidV.IsStr() {
		return nil, jsonrpc.ErrInvalidParams.AddData("msg", "oid required and should be string")
	}

	result := make([]map[string]interface{}, 0)

	err := s.cli.BulkWalk(oidV.Str(), func(dataUnit g.SnmpPDU) error {
		result = append(result, encodeDataUnit(dataUnit))
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func parsePDU(params objx.Map, k string) ([]g.SnmpPDU, error) {
	v := params.Get(k)
	if !v.IsObjxMapSlice() {
		return nil, jsonrpc.ErrInvalidParams.AddData("msg", k+" should be array")
	}

	data := make([]g.SnmpPDU, len(v.ObjxMapSlice()))

	for i, val := range v.ObjxMapSlice() {
		oidV := val.Get("oid")
		if !oidV.IsStr() {
			return nil, jsonrpc.ErrInvalidParams.AddData("msg", "oid should be string")
		}

		typ, err := getUint8(params, "type")
		if err != nil {
			return nil, err
		}

		data[i] = g.SnmpPDU{
			Name:  oidV.Str(),
			Type:  g.Asn1BER(typ),
			Value: params.Get("value").Data(),
		}
	}

	return data, nil
}

func (s Service) set(params objx.Map) (interface{}, error) {
	data, err := parsePDU(params, "pdus")
	if err != nil {
		return nil, err
	}

	res, err := s.cli.Set(data)
	if err != nil {
		return nil, err
	}

	return encodeSnmpPacket(res), nil
}

func (s Service) sendTrap(params objx.Map) (interface{}, error) { // nolint: funlen
	var trap g.SnmpTrap

	enterprise := params.Get("enterprise")
	if !enterprise.IsStr() {
		return nil, jsonrpc.ErrInvalidParams.AddData("msg", "enterprise should be string")
	}

	trap.Enterprise = enterprise.Str()

	agentAddr := params.Get("agent_address")
	if !agentAddr.IsStr() {
		return nil, jsonrpc.ErrInvalidParams.AddData("msg", "agent_address should be string")
	}

	trap.AgentAddress = agentAddr.Str()

	genericTrap, ok := params.Get("generic_trap").Data().(json.Number)
	if !ok {
		return nil, jsonrpc.ErrInvalidParams.AddData("msg", "generic_trap should be number")
	}

	gt, err := genericTrap.Int64()
	if err != nil {
		return nil, jsonrpc.ErrInvalidParams.AddData("msg", "generic_trap should be number")
	}

	trap.GenericTrap = int(gt)

	specificTrap, ok := params.Get("specific_trap").Data().(json.Number)
	if !ok {
		return nil, jsonrpc.ErrInvalidParams.AddData("msg", "specific_trap should be number")
	}

	st, err := specificTrap.Int64()
	if err != nil {
		return nil, jsonrpc.ErrInvalidParams.AddData("msg", "specific_trap should be number")
	}

	trap.SpecificTrap = int(st)

	timestamp, ok := params.Get("timestamp").Data().(json.Number)
	if !ok {
		return nil, jsonrpc.ErrInvalidParams.AddData("msg", "timestamp should be number")
	}

	ts, err := timestamp.Int64()
	if err != nil {
		return nil, jsonrpc.ErrInvalidParams.AddData("msg", "timestamp should be number")
	}

	trap.Timestamp = uint(ts)

	pdus, err := parsePDU(params, "variables")
	if err != nil {
		return nil, err
	}

	trap.Variables = pdus

	res, err := s.cli.SendTrap(trap)
	if err != nil {
		return nil, err
	}

	return encodeSnmpPacket(res), nil
}

func (s Service) Close() error {
	return s.cli.Conn.Close()
}
