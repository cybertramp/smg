package tuiadd

import (
	"fmt"
	"smg/cmd/comm"
	"smg/cmd/config"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	noStyle      = lipgloss.NewStyle()
)

const (
	fName = iota
	fIP
	fPort
	fUser
	fCertType
	fCert
	nFields
)

type model struct {
	focusIndex int
	inputs     []textinput.Model
	conns      []comm.Conn // existing entries (for validation)
	editIdx    int         // -1 when adding
	label      string
	result     comm.Conn
	submitted  bool
	err        error
}

func newModel(conns []comm.Conn, editIdx int) model {
	m := model{inputs: make([]textinput.Model, nFields), conns: conns, editIdx: editIdx, label: "Append"}

	placeholders := [nFields]string{
		"Name", "Ip(Url)", "Port (default 22)", "Id",
		"Cert type(0: agent / 1: PW / 2: keyfile)",
		"Cert(PW(blank: ask on connect) or keyfile path)",
	}
	for i := range m.inputs {
		t := textinput.New()
		t.Placeholder = placeholders[i]
		t.CharLimit = 256
		m.inputs[i] = t
	}

	if editIdx >= 0 {
		c := conns[editIdx]
		m.label = "Save"
		m.inputs[fName].SetValue(c.Name)
		m.inputs[fIP].SetValue(c.IP)
		m.inputs[fPort].SetValue(strconv.Itoa(c.Port))
		m.inputs[fUser].SetValue(c.User)
		m.inputs[fCertType].SetValue(strconv.Itoa(c.CertType))
		m.inputs[fCert].SetValue(c.Cert)
	}

	m.syncEcho()
	m.focus(0)
	return m
}

// syncEcho masks the Cert field only when it holds a password.
func (m *model) syncEcho() {
	if strings.TrimSpace(m.inputs[fCertType].Value()) == "1" {
		m.inputs[fCert].EchoMode = textinput.EchoPassword
	} else {
		m.inputs[fCert].EchoMode = textinput.EchoNormal
	}
}

func (m *model) focus(idx int) tea.Cmd {
	m.focusIndex = idx
	var cmd tea.Cmd
	for i := range m.inputs {
		if i == idx {
			cmd = m.inputs[i].Focus()
			m.inputs[i].PromptStyle = focusedStyle
			m.inputs[i].TextStyle = focusedStyle
			continue
		}
		m.inputs[i].Blur()
		m.inputs[i].PromptStyle = noStyle
		m.inputs[i].TextStyle = noStyle
	}
	return cmd
}

func (m model) build() (comm.Conn, error) {
	port := comm.DefaultPort
	if v := strings.TrimSpace(m.inputs[fPort].Value()); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return comm.Conn{}, fmt.Errorf("invalid port: %q", v)
		}
		port = p
	}
	certType, err := strconv.Atoi(strings.TrimSpace(m.inputs[fCertType].Value()))
	if err != nil {
		return comm.Conn{}, fmt.Errorf("invalid cert type: %q (0: agent / 1: PW / 2: keyfile)", m.inputs[fCertType].Value())
	}

	c := comm.Conn{
		Name:     strings.TrimSpace(m.inputs[fName].Value()),
		IP:       strings.TrimSpace(m.inputs[fIP].Value()),
		Port:     port,
		User:     strings.TrimSpace(m.inputs[fUser].Value()),
		CertType: certType,
		Cert:     m.inputs[fCert].Value(),
	}
	return c, config.Validate(c, m.conns, m.editIdx)
}

func (m model) Init() tea.Cmd { return textinput.Blink }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch s := msg.String(); s {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "tab", "shift+tab", "enter", "up", "down":
			if s == "enter" && m.focusIndex == nFields {
				c, err := m.build()
				if err != nil {
					m.err = err
					return m, nil
				}
				m.result, m.submitted = c, true
				return m, tea.Quit
			}

			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}
			if m.focusIndex > nFields {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = nFields
			}
			return m, m.focus(m.focusIndex)
		}
	}

	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	m.syncEcho()
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var b strings.Builder
	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		b.WriteRune('\n')
	}

	button := fmt.Sprintf("[ %s ]", blurredStyle.Render(m.label))
	if m.focusIndex == nFields {
		button = focusedStyle.Render(fmt.Sprintf("[ %s ]", m.label))
	}
	fmt.Fprintf(&b, "\n%s\n\n", button)

	if m.err != nil {
		b.WriteString(errStyle.Render(" "+m.err.Error()) + "\n\n")
	}
	b.WriteString(blurredStyle.Render(" (esc / ctrl+c to cancel)"))
	return b.String()
}

// Run shows the form. editIdx < 0 adds a new entry, otherwise edits conns[editIdx].
// ok is false if the user cancelled.
func Run(conns []comm.Conn, editIdx int) (conn comm.Conn, ok bool, err error) {
	final, err := tea.NewProgram(newModel(conns, editIdx)).Run()
	if err != nil {
		return comm.Conn{}, false, err
	}
	fm := final.(model)
	return fm.result, fm.submitted, nil
}
