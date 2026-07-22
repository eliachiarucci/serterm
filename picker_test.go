package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func key(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	default:
		r := []rune(s)[0]
		return tea.KeyPressMsg{Code: r, Text: s}
	}
}

func testPicker() pickerModel {
	m := newPickerModel()
	m.ports = []portInfo{
		{device: "/dev/cu.one"},
		{device: "/dev/cu.two"},
	}
	return m
}

func TestPickerCursorStaysInBounds(t *testing.T) {
	m := testPicker()

	m, _ = m.Update(key("up")) // already at the top
	if m.cursor != 0 {
		t.Errorf("cursor moved above the list: %d", m.cursor)
	}

	m, _ = m.Update(key("down"))
	m, _ = m.Update(key("down")) // already at the bottom
	if m.cursor != 1 {
		t.Errorf("cursor moved past the list: %d", m.cursor)
	}
}

func TestPickerCursorResetsWhenListShrinks(t *testing.T) {
	m := testPicker()
	m.cursor = 1

	m, _ = m.Update(portsMsg{ports: []portInfo{{device: "/dev/cu.only"}}})
	if m.cursor != 0 {
		t.Errorf("cursor should reset when out of bounds, got %d", m.cursor)
	}
}

func TestPickerBaudCyclesAndWraps(t *testing.T) {
	m := testPicker()
	if baudRates[m.baudIdx] != 115200 {
		t.Fatalf("default baud should be 115200, got %d", baudRates[m.baudIdx])
	}

	m, _ = m.Update(key("right"))
	if baudRates[m.baudIdx] != 230400 {
		t.Errorf("right should increase baud, got %d", baudRates[m.baudIdx])
	}

	m.baudIdx = 0
	m, _ = m.Update(key("left"))
	if m.baudIdx != len(baudRates)-1 {
		t.Errorf("left from first baud should wrap to last, got index %d", m.baudIdx)
	}
}

func TestPickerEnterSelectsDevice(t *testing.T) {
	m := testPicker()
	m.cursor = 1

	m, cmd := m.Update(key("enter"))
	if cmd == nil {
		t.Fatal("enter should emit a command")
	}
	msg, ok := cmd().(portSelectedMsg)
	if !ok {
		t.Fatalf("expected portSelectedMsg, got %T", cmd())
	}
	if msg.device != "/dev/cu.two" || msg.baud != 115200 {
		t.Errorf("unexpected selection: %+v", msg)
	}
}

func TestPickerEnterWithNoDevicesDoesNothing(t *testing.T) {
	m := newPickerModel()

	m, cmd := m.Update(key("enter"))
	if cmd != nil {
		t.Error("enter with an empty list should do nothing")
	}
}
