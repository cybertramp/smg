package tui_add_item

// A simple example demonstrating the use of multiple text input components
// from the Bubbles component library.

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"smg/cmd/comm"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	titleStyle          = lipgloss.NewStyle().MarginLeft(2)
	focusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle         = focusedStyle.Copy()
	noStyle             = lipgloss.NewStyle()
	helpStyle           = blurredStyle.Copy()
	cursorModeHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	focusedButton = focusedStyle.Copy().Render("[ Append ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Append"))
	flagAppend = false
)

type model struct {
	focusIndex int
	inputs     []textinput.Model
	cursorMode cursor.Mode
}

func initialModel() model {
	m := model{
		inputs: make([]textinput.Model, 6),
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 64

		switch i {
		case 0:
			t.Placeholder = "Name"
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Placeholder = "Ip(Url)"

			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 2:
			t.Placeholder = "Port"

			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 3:
			t.Placeholder = "Id"

			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 4:
			t.Placeholder = "Cert type(1: PW / 2: keyfile)"

			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 5:
			t.Placeholder = "Cert(PW or keyfile path)"

			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		}

		m.inputs[i] = t
	}

	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		// Set focus to next input
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Did the user press enter while the submit button was focused?
			// If so, exit.
			if s == "enter" && m.focusIndex == len(m.inputs) {
				flagAppend = true
				return m, tea.Quit
			}

			// Cycle indexes
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					// Set focused state
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
					continue
				}
				// Remove focused state
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		}
	}

	cmd := m.updateInputs(msg)

	return m, cmd
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m model) View() string {
	var b strings.Builder

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	b.WriteString(helpStyle.Render(" (ctrl+c to cancle add item)"))

	return b.String()
}

func (m model) UpdateJson(json_config comm.JsonData) comm.JsonData{
	props_name := m.inputs[0].Value()
	props_ip := m.inputs[1].Value()
	props_port, _ := strconv.Atoi(m.inputs[2].Value())
	props_user := m.inputs[3].Value()
	props_cert_type, _ := strconv.Atoi(m.inputs[4].Value())
	props_cert := m.inputs[5].Value()

	data := comm.Conn{
		Name: props_name,
		IP: props_ip,
		Port: props_port,
		User: props_user,
		Cert_Type: props_cert_type,
		Cert: props_cert,
	}

	json_config.JsonData = append(json_config.JsonData, data)
	
	return json_config
}

func JsonDataWriteToJsonFile(config_file_path string, json_config comm.JsonData){
	json_data, _ := json.MarshalIndent(json_config,"", "    ")
	
	err := ioutil.WriteFile(config_file_path, json_data, os.FileMode(0644))
	if err != nil {
		log.Println(err)
	}
}

func TuiRun(config_file_path string, json_config comm.JsonData){
	output := termenv.NewOutput(os.Stdout)
	output.ClearScreen()
	output.ShowCursor()
	
	fmt.Println("SMG: Run add item")
	m := initialModel()

	if _, err := tea.NewProgram(m).Run(); err != nil {
		log.Printf("Oops😓: %v", err)
		os.Exit(1)
	}

	if flagAppend{
		json_config = m.UpdateJson(json_config)
		JsonDataWriteToJsonFile(config_file_path, json_config)
	}
}