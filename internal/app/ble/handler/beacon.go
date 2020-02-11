package handler

import (
	"fmt"

	"github.com/Rightech/ric-edge/third_party/go-ble/ble"
)

var (
	preffixes = []string{"http://www.", "https://www.", "http://", "https://"}

	suffixes = []string{
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

	var beaconKind, beaconContent string

	if len(packet.ServiceData()) > 1 {
		panic("Service data length is " + string(len(packet.ServiceData())))
	}

	for _, serviceData := range packet.ServiceData() {

		eddystoneData := serviceData.Data
		typ := EddystoneBeacon(eddystoneData[0])

		switch typ {
		case EddystoneURL:
			beaconContent = getEddystoneURLParams(eddystoneData)
			beaconKind = "Eddystone URL"
		default:
			panic(fmt.Sprintf("Unsupported format: %x", typ))
		}

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
