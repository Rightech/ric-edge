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
	"fmt"
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

func modeToMode(mode string) (g.SnmpV3MsgFlags, error) {
	switch mode {
	case "authPriv":
		return g.AuthPriv, nil
	case "authNoPriv":
		return g.AuthNoPriv, nil
	case "NoauthNoPriv":
		return g.NoAuthNoPriv, nil
		//	case "Reportable":
		//		return g.Reportable, nil
	case "":
		return g.NoAuthNoPriv, nil
	default:
		return g.SnmpV3MsgFlags(^uint8(0)), errors.New("snmp: unknown security mode")
	}
}

func authprotToAuthProt(authProtocol string) (g.SnmpV3AuthProtocol, error) {
	switch authProtocol {
	case "MD5":
		return g.MD5, nil
	case "SHA":
		return g.SHA, nil
	case "":
		return g.MD5, nil
	default:
		return g.SnmpV3AuthProtocol(^uint8(0)), errors.New("snmp: unknown or unsupported authentication protocol")
	}
}

func privprotToPrivProt(privProtocol string) (g.SnmpV3PrivProtocol, error) {
	switch privProtocol {
	case "DES":
		return g.DES, nil
	case "AES":
		return g.AES, nil
		//	case "AES192":
		//		return g.AES192, nil
		//	case "AES256":
		//		return g.AES256, nil
	case "":
		return g.NoPriv, nil
		//	case "AES192C":
		//		return g.AES192C, nil
		//	case "AES256C":
		//		return g.AES256C, nil
	default:
		return g.SnmpV3PrivProtocol(^uint8(0)), errors.New("snmp: unknown or unsupported privacy protocol")
	}
}

func New(hostPort, community, version, mode, authProtocol, authKey, privProtocol, privKey,
	securityName string) (Service, error) {
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

	secMod, err := modeToMode(mode)
	if err != nil {
		return Service{}, err
	}

	authProt, err := authprotToAuthProt(authProtocol)
	if err != nil {
		return Service{}, err
	}

	privProt, err := privprotToPrivProt(privProtocol)
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
		SecurityModel:      g.UserSecurityModel,
		MaxOids:            g.MaxOids,
		Logger:             logger.New("info", log.DebugLevel),
		MsgFlags:           secMod,
		SecurityParameters: &g.UsmSecurityParameters{UserName: securityName,
			AuthenticationProtocol:   authProt,
			AuthenticationPassphrase: authKey,
			PrivacyProtocol:          privProt,
			PrivacyPassphrase:        privKey,
		},
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
		//	case "snmp-bulk-walk":
		//		res, err = s.bulkWalk(req.Params)
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

func IPtoHEX(ip string) (string, error) {
	s := strings.Split(ip, ".")

	a, err := strconv.ParseUint(s[0], 10, 32)
	if err != nil {
		return "", errors.New("invalid ip")
	}

	b, err := strconv.ParseUint(s[1], 10, 32)
	if err != nil {
		return "", errors.New("invalid ip")
	}

	c, err := strconv.ParseUint(s[2], 10, 32)
	if err != nil {
		return "", errors.New("invalid ip")
	}

	d, err := strconv.ParseUint(s[3], 10, 32)
	if err != nil {
		return "", errors.New("invalid ip")
	}

	if a > 255 || b > 255 || c > 255 || d > 255 {
		return "", errors.New("invalid ip")
	}

	h1 := fmt.Sprintf("%02x", a)
	h2 := fmt.Sprintf("%02x", b)
	h3 := fmt.Sprintf("%02x", c)
	h4 := fmt.Sprintf("%02x", d)

	return h1 + h2 + h3 + h4, nil
}

func (s Service) set(params objx.Map) (interface{}, error) {
	oidV := params.Get("oid")
	if !oidV.IsStr() {
		return nil, jsonrpc.ErrInvalidParams.AddData("msg", "oid required and should be string")
	}

	typ, err := getUint8(params, "type")
	if err != nil {
		return nil, err
	}

	switch typ {
	case 4: // STRING
		res, err := s.cli.Set([]g.SnmpPDU{{
			Name:  oidV.Str(),
			Type:  g.OctetString,
			Value: params.Get("value").Str(),
		}})
		if err != nil {
			return nil, err
		}

		return encodeSnmpPacket(res), nil

	case 2: // INTEGER
		v, err := strconv.Atoi(params.Get("value").Str())
		if err != nil {
			return nil, err
		}

		res, err := s.cli.Set([]g.SnmpPDU{{
			Name:  oidV.Str(),
			Type:  g.Integer,
			Value: v,
		}})
		if err != nil {
			return nil, err
		}

		return encodeSnmpPacket(res), nil

	case 66: // Gauge32
		v, err := strconv.ParseUint(params.Get("value").Str(), 10, 64)
		if err != nil {
			return nil, err
		}

		res, err := s.cli.Set([]g.SnmpPDU{{
			Name:  oidV.Str(),
			Type:  g.Gauge32,
			Value: uint(v),
		}})
		if err != nil {
			return nil, err
		}

		return encodeSnmpPacket(res), nil

	case 64: // IPAddress
		v, err := IPtoHEX(params.Get("value").Str())
		if err != nil {
			return nil, err
		}

		res, err := s.cli.Set([]g.SnmpPDU{{
			Name:  oidV.Str(),
			Type:  g.IPAddress,
			Value: v,
		}})
		if err != nil {
			return nil, err
		}

		return encodeSnmpPacket(res), nil
	case 6: // OID
		res, err := s.cli.Set([]g.SnmpPDU{{
			Name:  oidV.Str(),
			Type:  g.ObjectIdentifier,
			Value: params.Get("value").Str(),
		}})
		if err != nil {
			return nil, err
		}

		return encodeSnmpPacket(res), nil

	case 67: // Timeticks
		v, err := strconv.ParseUint(params.Get("value").Str(), 10, 32)
		if err != nil {
			return nil, err
		}

		res, err := s.cli.Set([]g.SnmpPDU{{
			Name:  oidV.Str(),
			Type:  g.TimeTicks,
			Value: uint32(v),
		}})
		if err != nil {
			return nil, err
		}

		return encodeSnmpPacket(res), nil

	case 65: // Counter32
		v, err := strconv.ParseUint(params.Get("value").Str(), 10, 64)
		if err != nil {
			return nil, err
		}

		res, err := s.cli.Set([]g.SnmpPDU{{
			Name:  oidV.Str(),
			Type:  g.Counter32,
			Value: uint(v),
		}})
		if err != nil {
			return nil, err
		}

		return encodeSnmpPacket(res), nil

	case 70: // Counter64
		v, err := strconv.ParseUint(params.Get("value").Str(), 10, 64)
		if err != nil {
			return nil, err
		}

		res, err := s.cli.Set([]g.SnmpPDU{{
			Name:  oidV.Str(),
			Type:  g.Counter64,
			Value: v,
		}})
		if err != nil {
			return nil, err
		}

		return encodeSnmpPacket(res), nil

	default:
		return nil, jsonrpc.ErrInvalidParams.AddData("msg", "unknown or unsupported type")
	}
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
