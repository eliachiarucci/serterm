package main

import (
	"testing"

	"go.bug.st/serial/enumerator"
)

func TestBuildPortList(t *testing.T) {
	details := []*enumerator.PortDetails{
		{Name: "/dev/cu.Bluetooth-Incoming-Port"},
		{Name: "/dev/tty.usbmodem1101", IsUSB: true, VID: "2341", PID: "0043", Product: "Arduino Uno"},
		{Name: "/dev/cu.usbmodem1101", IsUSB: true, VID: "2341", PID: "0043", Product: "Arduino Uno"},
		{Name: "/dev/cu.usbserial-0001", IsUSB: true, VID: "10C4", PID: "EA60"}, // no product name
	}

	ports := buildPortList(details, "darwin")

	// tty.* duplicates are hidden on macOS.
	for _, p := range ports {
		if p.device == "/dev/tty.usbmodem1101" {
			t.Errorf("tty.* duplicate should be filtered out, got %v", p)
		}
	}
	if len(ports) != 3 {
		t.Fatalf("expected 3 ports, got %d: %v", len(ports), ports)
	}

	// USB devices come first, non-USB last.
	if ports[0].device != "/dev/cu.usbmodem1101" || ports[1].device != "/dev/cu.usbserial-0001" {
		t.Errorf("USB devices should be sorted first, got %v", ports)
	}
	if ports[2].device != "/dev/cu.Bluetooth-Incoming-Port" {
		t.Errorf("non-USB device should be last, got %v", ports)
	}

	// Descriptions: product name when known, fallback otherwise, VID:PID always.
	if ports[0].description != "Arduino Uno [2341:0043]" {
		t.Errorf("unexpected description: %q", ports[0].description)
	}
	if ports[1].description != "USB serial device [10C4:EA60]" {
		t.Errorf("unexpected fallback description: %q", ports[1].description)
	}

	// On other platforms tty.* is a real device and must be kept.
	if got := buildPortList(details, "linux"); len(got) != 4 {
		t.Errorf("expected 4 ports on linux, got %d", len(got))
	}
}
