package tui_main

import (
	"fmt"
	"io"
	"log"
	"os"
	"smg/cmd/comm"
	"smg/cmd/ssh"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)

	conn_target       = comm.Conn{}
	flag_conn		   = false
)

type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprintf(w, "%s", fn(str))
}

type Tui struct {
	Data comm.JsonData
    List     list.Model
	Items    []item
	Choice   string
	ChoiceIdx int
	Quitting bool
}

func (m Tui) Init() tea.Cmd {
	return nil
}

func (m Tui) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.List.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+d", "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.List.SelectedItem().(item)
			if ok {
				m.ChoiceIdx = m.List.Index()
				m.Choice = string(i)
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return m, cmd
}

func (m Tui) View() string {

	output := termenv.NewOutput(os.Stdout)
	
	if m.Choice != "" {
		output.ClearScreen()
		output.ShowCursor()		// Enable Cursor in shell
		
		flag_conn = true
		conn_target = m.Data.JsonData[m.ChoiceIdx]
	}

	if m.Quitting {
		output.ClearScreen()
		return quitTextStyle.Render("Bye! 👋(Ctrl+c)")
	}

	return "\n" + m.List.View()
}

func ConvertTuiListFromArr(json_conn comm.JsonData) []list.Item{

	items := []list.Item{}

    for _, d := range json_conn.JsonData {
		items = append(items, item(d.Name))
    }

    return items
}

func TuiRun(config_file_path string, json_config comm.JsonData) {

	if len(json_config.JsonData) <=0{
		log.Fatalln("Connection data not exist! Please run 'smg add' or add manual!")
		log.Fatalf("(%s)\n", config_file_path)
	}

    items := ConvertTuiListFromArr(json_config)

	const defaultWidth = 20

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Choose where to connect with ssh? 🚲"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := Tui{Data: json_config, List: l}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		log.Fatalf("Oops😓: %v", err)
		os.Exit(1)
	}

	output := termenv.NewOutput(os.Stdout)

	if flag_conn{
		conn_err := ssh.RunSSH(conn_target)
		if conn_err != nil {
			output.ClearScreen()
		}
	}
}