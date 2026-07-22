package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"go.bug.st/serial"
)

// terminalChrome is the number of fixed lines around the viewport:
// header, input line, and footer with key hints.
const terminalChrome = 3

// maxContentBytes caps the scrollback buffer so long sessions stay light.
const maxContentBytes = 256 * 1024

// serialDataMsg carries bytes read from the device.
type serialDataMsg []byte

// serialClosedMsg reports that reading stopped (unplugged device, I/O error).
type serialClosedMsg struct{ err error }

// terminalModel is the streaming screen: device output on top, input line at the bottom.
type terminalModel struct {
	port   serial.Port
	device string
	baud   int

	viewport viewport.Model
	input    textinput.Model
	focusCmd tea.Cmd // cursor blink command captured at construction
	content  string

	reads        chan tea.Msg  // filled by the reader goroutine
	done         chan struct{} // closed when this session ends
	closed       bool          // port has been closed by us
	disconnected bool          // reading stopped on its own
	pendingCR    bool          // last read chunk ended with \r

	width  int
	height int
}

func newTerminalModel(port serial.Port, device string, baud, width, height int) terminalModel {
	input := textinput.New()
	input.Prompt = "❯ "
	input.Placeholder = "type a message, enter to send"
	input.SetWidth(inputWidth(width, input.Prompt))
	focusCmd := input.Focus()

	m := terminalModel{
		port:     port,
		device:   device,
		baud:     baud,
		viewport: viewport.New(viewport.WithWidth(width), viewport.WithHeight(max(height-terminalChrome, 1))),
		input:    input,
		focusCmd: focusCmd,
		reads:    make(chan tea.Msg),
		done:     make(chan struct{}),
		width:    width,
		height:   height,
	}
	go m.readLoop()
	return m
}

func inputWidth(termWidth int, prompt string) int {
	return max(termWidth-lipgloss.Width(prompt), 10)
}

// readLoop runs in a goroutine and forwards device output to the UI.
func (m terminalModel) readLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := m.port.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			select {
			case m.reads <- serialDataMsg(data):
			case <-m.done:
				return
			}
		}
		if err != nil {
			select {
			case m.reads <- serialClosedMsg{err: err}:
			case <-m.done:
			}
			return
		}
	}
}

// waitForSerial is the command that delivers the next reader message.
func (m terminalModel) waitForSerial() tea.Cmd {
	return func() tea.Msg {
		select {
		case msg := <-m.reads:
			return msg
		case <-m.done:
			return nil
		}
	}
}

func (m terminalModel) Init() tea.Cmd {
	return tea.Batch(m.focusCmd, m.waitForSerial())
}

func (m terminalModel) Update(msg tea.Msg) (terminalModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.viewport.SetWidth(msg.Width)
		m.viewport.SetHeight(max(msg.Height-terminalChrome, 1))
		m.input.SetWidth(inputWidth(msg.Width, m.input.Prompt))
		m.refreshViewport()
		m.viewport.GotoBottom()
		return m, nil

	case serialDataMsg:
		m.append(m.normalize(string(msg)))
		return m, m.waitForSerial()

	case serialClosedMsg:
		if !m.closed {
			m.disconnected = true
			m.append(noticeStyle.Render(fmt.Sprintf("── connection lost (%v) · ctrl+d to go back ──", msg.err)) + "\n")
		}
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			m.shutdown()
			return m, tea.Quit
		case "ctrl+d":
			m.shutdown()
			return m, func() tea.Msg { return backToPickerMsg{} }
		case "enter":
			return m.send(), nil
		case "ctrl+l":
			m.content = ""
			m.refreshViewport()
			m.viewport.GotoTop()
			return m, nil
		// up/down is what the mouse wheel produces in alternate scroll
		// mode, which the emulator uses because mouse reporting is off.
		case "up", "down", "pgup", "pgdown", "home", "end":
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// send writes the input line to the device and echoes it in the stream.
func (m terminalModel) send() terminalModel {
	text := m.input.Value()
	if text == "" || m.disconnected || m.closed {
		return m
	}
	if _, err := m.port.Write([]byte(text + "\n")); err != nil {
		m.append(noticeStyle.Render(fmt.Sprintf("── write failed (%v) ──", err)) + "\n")
		return m
	}
	m.append(sentStyle.Render("→ "+text) + "\n")
	m.input.Reset()
	return m
}

// normalize converts device line endings to \n. A \r\n pair can arrive split
// across two read chunks, so a \r at the end of a chunk is remembered and the
// \n opening the next chunk is swallowed instead of becoming a second newline.
func (m *terminalModel) normalize(s string) string {
	if m.pendingCR && strings.HasPrefix(s, "\n") {
		s = s[1:]
	}
	m.pendingCR = strings.HasSuffix(s, "\r")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

// append adds text to the stream, trims old content, and keeps the view
// pinned to the bottom unless the user has scrolled up.
func (m *terminalModel) append(s string) {
	atBottom := m.viewport.AtBottom()
	m.content += s
	if len(m.content) > maxContentBytes {
		cut := len(m.content) - maxContentBytes
		if i := strings.IndexByte(m.content[cut:], '\n'); i >= 0 {
			cut += i + 1
		}
		m.content = m.content[cut:]
	}
	m.refreshViewport()
	if atBottom {
		m.viewport.GotoBottom()
	}
}

// refreshViewport pushes the content into the viewport, hard-wrapping lines
// that are longer than the terminal is wide.
func (m *terminalModel) refreshViewport() {
	content := m.content
	if m.width > 0 {
		content = ansi.Hardwrap(content, m.width, true)
	}
	m.viewport.SetContent(content)
}

// shutdown stops the reader goroutine and closes the port.
func (m *terminalModel) shutdown() {
	if m.closed {
		return
	}
	m.closed = true
	close(m.done)
	m.port.Close()
}

func (m terminalModel) View() string {
	status := fmt.Sprintf(" %s @ %d baud", m.device, m.baud)
	if m.disconnected {
		status += " · disconnected"
	}
	pad := max(m.width-lipgloss.Width(status), 0)
	header := headerStyle.Render(status + strings.Repeat(" ", pad))

	footer := dimStyle.Render(" enter: send · ctrl+l: clear · ctrl+d: devices · pgup/pgdn: scroll · ctrl+c: quit")

	return header + "\n" + m.viewport.View() + "\n" + m.input.View() + "\n" + footer
}
