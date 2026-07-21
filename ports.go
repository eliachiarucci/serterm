package main

import (
	"runtime"
	"sort"
	"strings"

	"go.bug.st/serial/enumerator"
)

type portInfo struct {
	device      string
	description string
}

// listPorts returns the serial devices available on the system.
//
// On macOS every device shows up twice, as /dev/cu.* and /dev/tty.*.
// The cu.* entry is the right one for initiating connections (it does not
// wait for a carrier signal), so the tty.* duplicates are hidden.
func listPorts() ([]portInfo, error) {
	details, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, err
	}
	return buildPortList(details, runtime.GOOS), nil
}

// buildPortList turns raw enumerator results into the list shown to the user.
func buildPortList(details []*enumerator.PortDetails, goos string) []portInfo {
	var ports []portInfo
	for _, d := range details {
		if goos == "darwin" && strings.HasPrefix(d.Name, "/dev/tty.") {
			continue
		}
		desc := ""
		if d.IsUSB {
			desc = d.Product
			if desc == "" {
				desc = "USB serial device"
			}
			desc += " [" + d.VID + ":" + d.PID + "]"
		}
		ports = append(ports, portInfo{device: d.Name, description: desc})
	}

	// USB devices first (most likely what the user wants), then alphabetical.
	sort.SliceStable(ports, func(i, j int) bool {
		iUSB, jUSB := ports[i].description != "", ports[j].description != ""
		if iUSB != jUSB {
			return iUSB
		}
		return ports[i].device < ports[j].device
	})
	return ports
}
