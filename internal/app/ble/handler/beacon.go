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
	// Eddystone beacon has such service uuid
	// which is always exposed
	eddystoneService string = "feaa"
)

// Get eddystone params
func getEddystoneParams(packet ble.Advertisement) (string, string) {

	var beaconKind, beaconContent string

	for _, serviceData := range packet.ServiceData() {
		eddystoneData := serviceData.Data
		beaconType := eddystoneData[:1]
		urlPrefix := eddystoneData[2:3]
		urlContent := string(eddystoneData[3 : len(eddystoneData)-1])
		urlSuffix := eddystoneData[len(eddystoneData)-1]
		preffix := preffixes[urlPrefix[0]]
		suffix := suffixes[urlSuffix]
		beaconKind = getEddystoneBeaconKind(int8(beaconType[0]))
		beaconContent = preffix + urlContent + suffix

	}

	return beaconKind, beaconContent
}

// get type of Eddystone beacon
func getEddystoneBeaconKind(beaconType int8) string {
	beacon := ""
	switch beaconType {
	case 0x00:
		beacon = "UID"
	case 0x10:
		beacon = "URL"
	case 0x20:
		beacon = "TLM"
	default:
		beacon = "undefined"
	}

	return beacon
}
