package handler

import (
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
	eddystoneUID     string = "Eddystone UID"
	eddystoneURL     string = "Eddystone URL"
	eddystoneTLM     string = "Eddystone TLM"
)

type beacon struct {
	BeaconType    string `json:"beaconType"`
	BeaconContent string `json:"beaconContent"`
}

// Get eddystone params
func getEddystoneParams(packet ble.Advertisement) *beacon {

	var beaconKind, beaconContent string

	for _, serviceData := range packet.ServiceData() {
		eddystoneData := serviceData.Data
		urlPrefix := eddystoneData[2:3]
		urlContent := string(eddystoneData[3 : len(eddystoneData)-1])
		urlSuffix := eddystoneData[len(eddystoneData)-1]
		preffix := preffixes[urlPrefix[0]]
		suffix := suffixes[urlSuffix]
		beaconKind = getEddystoneBeaconKind(eddystoneData[0])
		beaconContent = preffix + urlContent + suffix
	}

	return &beacon{
		BeaconType:    beaconKind,
		BeaconContent: beaconContent,
	}
}

// get type of Eddystone beacon
func getEddystoneBeaconKind(beaconType byte) string {
	switch beaconType {
	case 0x00:
		return eddystoneUID
	case 0x10:
		return eddystoneURL
	case 0x20:
		return eddystoneTLM
	default:
		return "undefined"
	}

}
