package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var baudRates = []int{9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600}

const defaultBaudIndex = 4 // 115200

// pickerModel is the device selection screen.
type pickerModel struct {
	ports   []portInfo
	cursor  int
	baudIdx int
	err     error
}

func newPickerModel() pickerModel {
	return pickerModel{baudIdx: defaultBaudIndex}
}

// portsMsg carries the result of a port scan.
type portsMsg struct {
	ports []portInfo
	err   error
}

func refreshPorts() tea.Msg {
	ports, err := listPorts()
	return portsMsg{ports: ports, err: err}
}

func (m pickerModel) Init() tea.Cmd {
	return refreshPorts
}

func (m pickerModel) Update(msg tea.Msg) (pickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case portsMsg:
		m.ports, m.err = msg.ports, msg.err
		if m.cursor >= len(m.ports) {
			m.cursor = 0
		}

	case openFailedMsg:
		m.err = msg.err

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "ctrl+d":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.ports)-1 {
				m.cursor++
			}
		case "left", "h":
			m.baudIdx = (m.baudIdx + len(baudRates) - 1) % len(baudRates)
		case "right", "l":
			m.baudIdx = (m.baudIdx + 1) % len(baudRates)
		case "r":
			return m, refreshPorts
		case "enter":
			if len(m.ports) > 0 {
				m.err = nil
				selected := portSelectedMsg{
					device: m.ports[m.cursor].device,
					baud:   baudRates[m.baudIdx],
				}
				return m, func() tea.Msg { return selected }
			}
		}
	}
	return m, nil
}

func (m pickerModel) View() string {
	var b strings.Builder

	b.WriteString("\n  " + titleStyle.Render("SerTerm") + dimStyle.Render("  ·  select a device") + "\n\n")

	if len(m.ports) == 0 {
		b.WriteString(dimStyle.Render("    no serial devices found — plug one in and press r to refresh") + "\n")
	}
	for i, p := range m.ports {
		device := p.device
		prefix := "    "
		if i == m.cursor {
			prefix = "  " + cursorStyle.Render("▸ ")
			device = selectedStyle.Render(device)
		}
		b.WriteString(prefix + device)
		if p.description != "" {
			b.WriteString("  " + dimStyle.Render(p.description))
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("\n  baud rate: %s\n", selectedStyle.Render(fmt.Sprint(baudRates[m.baudIdx]))))

	if m.err != nil {
		b.WriteString("\n  " + errorStyle.Render(m.err.Error()) + "\n")
	}

	b.WriteString("\n  " + dimStyle.Render("↑/↓ select · ←/→ baud · enter connect · r refresh · q quit") + "\n")
	return b.String()
}
