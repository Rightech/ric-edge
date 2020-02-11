/**
 * Copyright 2020 Rightech IoT. All Rights Reserved.
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

	"github.com/Rightech/ric-edge/third_party/go-ble/ble"
)

var (
	preffixes = []string{"http://www.", "https://www.", "http://", "https://"} // nolint: gochecknoglobals

	suffixes = []string{ // nolint: gochecknoglobals
		".com/",
		".org/",
		".edu/",
		".net/",
		".info/",
		".biz/",
		".gov/",
		".com",
		".org",
		".edu",
		".net",
		".info",
		".biz",
		".gov",
	}
)

const (
	eddystoneService string = "feaa"
)

// EddystoneBeacon is type for common eddystone beacons
type EddystoneBeacon byte

// uses for different parsing funcs
const (
	EddystoneUID EddystoneBeacon = 0x00
	EddystoneURL EddystoneBeacon = 0x10
	EddystoneTLM EddystoneBeacon = 0x20
)

type beacon struct {
	BeaconType    string `json:"beaconType"`
	BeaconContent string `json:"beaconContent"`
}

// Get eddystone params
func getEddystoneParams(packet ble.Advertisement) *beacon {
	if len(packet.ServiceData()) == 0 {
		return nil
	}

	var (
		beaconKind    = ""
		beaconContent = ""
	)

	if len(packet.ServiceData()) > 1 {
		panic(fmt.Sprintf("Service data length is %v", len(packet.ServiceData())))
	}

	serviceData := packet.ServiceData()[0].Data
	typ := EddystoneBeacon(serviceData[0])

	switch typ {
	case EddystoneURL:
		beaconContent = getEddystoneURLParams(serviceData)
		beaconKind = "Eddystone URL"
	default:
		panic(fmt.Sprintf("Unsupported format: %x", typ))
	}

	return &beacon{
		BeaconType:    beaconKind,
		BeaconContent: beaconContent,
	}
}

func getEddystoneURLParams(packet []byte) string {
	urlPrefix := packet[2:3]
	urlContent := string(packet[3 : len(packet)-1])
	urlSuffix := packet[len(packet)-1]
	preffix := preffixes[urlPrefix[0]]
	suffix := suffixes[urlSuffix]
	beaconContent := preffix + urlContent + suffix

	return beaconContent
}
