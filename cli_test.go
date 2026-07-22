package main

import (
	"testing"
	"time"
)

func TestIsUpToDate(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{name: "same version", current: "1.2.3", latest: "v1.2.3", want: true},
		{name: "tag without v prefix", current: "1.2.3", latest: "1.2.3", want: true},
		{name: "older than latest", current: "1.2.2", latest: "v1.2.3", want: false},
		{name: "dev build", current: "dev", latest: "v1.2.3", want: false},
		{name: "empty tag", current: "1.2.3", latest: "", want: false},
		{name: "bare v tag", current: "1.2.3", latest: "v", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUpToDate(tt.current, tt.latest); got != tt.want {
				t.Fatalf("isUpToDate(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

func TestParseOpenArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		device  string
		flags   []string
		wantErr bool
	}{
		{name: "device only", args: []string{"/dev/cu.usb"}, device: "/dev/cu.usb"},
		{name: "device then flags", args: []string{"/dev/cu.usb", "-b", "9600"}, device: "/dev/cu.usb", flags: []string{"-b", "9600"}},
		{name: "no device", args: nil, wantErr: true},
		{name: "flag before device", args: []string{"-b", "9600", "/dev/cu.usb"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device, flags, err := parseOpenArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseOpenArgs(%v): expected error, got none", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOpenArgs(%v): unexpected error: %v", tt.args, err)
			}
			if device != tt.device {
				t.Fatalf("parseOpenArgs(%v) device = %q, want %q", tt.args, device, tt.device)
			}
			if len(flags) != len(tt.flags) {
				t.Fatalf("parseOpenArgs(%v) flags = %v, want %v", tt.args, flags, tt.flags)
			}
			for i := range flags {
				if flags[i] != tt.flags[i] {
					t.Fatalf("parseOpenArgs(%v) flags = %v, want %v", tt.args, flags, tt.flags)
				}
			}
		})
	}
}

func TestParseCloseAfter(t *testing.T) {
	tests := []struct {
		name     string
		secs     float64
		duration time.Duration
		wantErr  bool
	}{
		{name: "not given", secs: 0, duration: 0},
		{name: "whole seconds", secs: 5, duration: 5 * time.Second},
		{name: "fractional seconds", secs: 0.5, duration: 500 * time.Millisecond},
		{name: "max seconds", secs: 60, duration: 60 * time.Second},
		{name: "negative seconds", secs: -3, wantErr: true},
		{name: "over max seconds", secs: 61, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := parseCloseAfter(tt.secs)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseCloseAfter(%v): expected error, got none", tt.secs)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCloseAfter(%v): unexpected error: %v", tt.secs, err)
			}
			if duration != tt.duration {
				t.Fatalf("parseCloseAfter(%v) = %v, want %v", tt.secs, duration, tt.duration)
			}
		})
	}
}
