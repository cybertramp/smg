package tuimain

import (
	"fmt"
	"io"
	"smg/cmd/comm"

	"github.com/charmbracelet/bubbles/key"
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
	errStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

// Action is what the user asked for on the selected entry.
type Action int

const (
	ActionCancel  Action = iota
	ActionConnect        // enter
	ActionEdit           // e
	ActionDelete         // d (after confirmation)
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
	action   Action
	actions  bool   // enable e/d keys
	confirm  string // non-empty: waiting for y/n to delete this name
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		if m.confirm != "" {
			if k := msg.String(); k == "y" || k == "Y" {
				m.action = ActionDelete
				return m, tea.Quit
			}
			m.confirm = "" // anything else cancels
			return m, nil
		}
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
				m.action = ActionConnect
			}
			return m, tea.Quit
		case "e", "d":
			if filtering || !m.actions {
				break
			}
			i, ok := m.list.SelectedItem().(item)
			if !ok {
				return m, nil
			}
			m.selected = i.idx
			if msg.String() == "e" {
				m.action = ActionEdit
				return m, tea.Quit
			}
			m.confirm = i.name
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.action != ActionCancel {
		return ""
	}
	if len(m.list.Items()) == 0 {
		return "\n" + titleStyle.Render(m.list.Title) + "\n\n" +
			itemStyle.Render("No items (Please add host)") + "\n\n" +
			helpStyle.Render("q quit") + "\n"
	}
	if m.confirm != "" {
		return "\n" + errStyle.Render(fmt.Sprintf("  Delete '%s'? (y/N)", m.confirm)) + "\n"
	}
	return "\n" + m.list.View()
}

// Pick shows a filterable list ('/') and returns the index of the chosen entry; ok is false if cancelled.
func Pick(title string, conns []comm.Conn) (idx int, ok bool, err error) {
	idx, act, err := pick(title, conns, false)
	return idx, act == ActionConnect, err
}

// PickAction is Pick plus 'e' (edit) and 'd' (delete, confirmed with y) on the selected entry.
func PickAction(title string, conns []comm.Conn) (idx int, act Action, err error) {
	return pick(title, conns, true)
}

func pick(title string, conns []comm.Conn, actions bool) (int, Action, error) {
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
	if actions {
		extra := []key.Binding{
			key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		}
		l.AdditionalShortHelpKeys = func() []key.Binding { return extra }
		l.AdditionalFullHelpKeys = func() []key.Binding { return extra }
	}

	ClearScreen()
	final, err := tea.NewProgram(model{list: l, actions: actions}).Run()
	ClearScreen()
	if err != nil {
		return 0, ActionCancel, err
	}
	fm := final.(model)
	return fm.selected, fm.action, nil
}

// ClearScreen clears the terminal (before/after a TUI and around the ssh session).
func ClearScreen() {
	out := termenv.DefaultOutput()
	out.ClearScreen()
	out.ShowCursor()
}
