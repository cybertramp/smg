package tuimain

import (
	"fmt"
	"io"
	"smg/cmd/comm"

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
)

// item keeps the index into the original slice, so filtering does not change the result.
type item struct {
	idx   int
	label string
	name  string
}

func (i item) FilterValue() string { return i.name }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}
	if index == m.Index() {
		_, _ = fmt.Fprint(w, selectedItemStyle.Render(fmt.Sprintf("> %d. %s", index+1, i.label)))
		return
	}
	_, _ = fmt.Fprint(w, itemStyle.Render(fmt.Sprintf("%d. %s", index+1, i.label)))
}

type model struct {
	list     list.Model
	selected int
	done     bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		filtering := m.list.FilterState() == list.Filtering
		switch msg.String() {
		case "ctrl+c", "ctrl+d":
			return m, tea.Quit
		case "q":
			if !filtering {
				return m, tea.Quit
			}
		case "enter":
			if filtering {
				break // let the list accept the filter first
			}
			if i, ok := m.list.SelectedItem().(item); ok {
				m.selected = i.idx
				m.done = true
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.done {
		return ""
	}
	return "\n" + m.list.View()
}

// Pick shows a filterable list ('/') and returns the index of the chosen entry; ok is false if cancelled.
func Pick(title string, conns []comm.Conn) (idx int, ok bool, err error) {
	items := make([]list.Item, len(conns))
	for i, c := range conns {
		items[i] = item{idx: i, name: c.Name, label: fmt.Sprintf("%s  (%s@%s:%d)", c.Name, c.User, c.IP, c.Port)}
	}

	l := list.New(items, itemDelegate{}, 20, listHeight)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	final, err := tea.NewProgram(model{list: l}).Run()
	if err != nil {
		return 0, false, err
	}

	fm := final.(model)
	if !fm.done {
		return 0, false, nil
	}

	return fm.selected, true, nil
}

// ClearScreen resets the terminal before handing it over to the ssh session.
func ClearScreen() {
	out := termenv.DefaultOutput()
	out.ClearScreen()
	out.ShowCursor()
}
