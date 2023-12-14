package main

import (
	"errors"
	"fmt"
	"net/netip"
	"os"

	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
)

var errBadExpansion = errors.New("failed to expand variable (empty value)")

type deviceRejectedData struct {
	name    string
	device  protocol.DeviceID
	address netip.Addr
}

func getDeviceRejectedData(ev events.Event) (*deviceRejectedData, error) {
	dataMap, ok := ev.Data.(map[string]any)
	if !ok {
		return nil, errMalformedEvent
	}

	var res deviceRejectedData
	if name, ok := dataMap["name"].(string); !ok {
		return nil, errMalformedEvent
	} else {
		res.name = name
	}
	if devStr, ok := dataMap["device"].(string); !ok {
		return nil, errMalformedEvent
	} else if dev, err := protocol.DeviceIDFromString(devStr); err != nil {
		return nil, errMalformedEvent
	} else {
		res.device = dev
	}
	if addrStr, ok := dataMap["address"].(string); !ok {
		return nil, errMalformedEvent
	} else if addr, err := netip.ParseAddrPort(addrStr); err != nil {
		return nil, errMalformedEvent
	} else {
		res.address = addr.Addr()
	}

	return &res, nil
}

func replaceVariables(s string, d *deviceRejectedData) (string, error) {
	var err error
	res := os.Expand(s, func(key string) string {
		switch key {
		case "device":
			v := d.device.String()
			if v == "" {
				err = fmt.Errorf("%w: %s", errBadExpansion, key)
			}
			return v
		case "name":
			v := d.name
			if v == "" {
				err = fmt.Errorf("%w: %s", errBadExpansion, key)
			}
			return v
		case "address":
			v := d.address.String()
			if v == "" {
				err = fmt.Errorf("%w: %s", errBadExpansion, key)
			}
			return v
		}
		err = fmt.Errorf("%w: %s", errBadExpansion, key)
		return ""
	})
	return res, err
}
