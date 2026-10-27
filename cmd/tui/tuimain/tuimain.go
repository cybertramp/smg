package tuimain

import (
	"fmt"
	"io"
	"smg/cmd/comm"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
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
	ActionConnect        // enter on a connection
	ActionEdit           // e (a pod is renamed)
	ActionDelete         // d (after confirmation; a pod is deleted with its content)
	ActionAdd            // a
	ActionMove           // m (connection -> other pod)
	ActionNewPod         // p
)

// State is where Browse starts: the pod being viewed and an optional cursor hint.
type State struct {
	Path   []string
	HasSel bool // put the cursor on pod/conn number SelIdx of Path
	SelPod bool
	SelIdx int
	Notice string // error/info line shown under the list
}

// Result is the outcome of Browse. Browse is the pod being viewed when it ended;
// Path/Idx/IsPod locate the selected entry (Path is the pod that contains it).
type Result struct {
	Action Action
	Browse []string
	Path   []string
	Idx    int
	IsPod  bool
}

type kind int

const (
	kConn kind = iota
	kPod
	kUp
)

// item keeps its location in the tree, so filtering and sorting never change the result.
type item struct {
	kind  kind
	path  []string // pod containing it
	idx   int
	label string
	name  string // search text
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

func connLabel(c comm.Conn) string {
	label := fmt.Sprintf("%s  (%s@%s:%d)", c.Name, c.User, c.IP, c.Port)
	if t := ago(c.LastUsed); t != "" {
		label += "  · " + t
	}
	return label
}

// levelItems lists the pods (file order) and then the connections (most recently used first) of path.
func levelItems(data comm.JsonData, path []string) []list.Item {
	conns, pods, _ := data.Level(path)
	var items []list.Item
	if len(path) > 0 {
		items = append(items, item{kind: kUp, path: path, label: ".. (up)"})
	}
	for i, p := range *pods {
		items = append(items, item{kind: kPod, path: path, idx: i, name: p.Name,
			label: fmt.Sprintf("📁 %s/  (%d)", p.Name, p.Count())})
	}
	order := make([]int, len(*conns))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return (*conns)[order[a]].LastUsed > (*conns)[order[b]].LastUsed })
	for _, i := range order {
		c := (*conns)[i]
		items = append(items, item{kind: kConn, path: path, idx: i, name: c.Name, label: connLabel(c)})
	}
	return items
}

// searchItems lists every connection of every pod, so a search spans the whole tree.
func searchItems(data comm.JsonData) []list.Item {
	entries := data.Flatten()
	sort.SliceStable(entries, func(a, b int) bool { return entries[a].Conn.LastUsed > entries[b].Conn.LastUsed })
	items := make([]list.Item, len(entries))
	for n, e := range entries {
		where := comm.JoinPath(e.Path)
		label := connLabel(e.Conn)
		if where != "" {
			label = "📁 " + where + " › " + label
		}
		items[n] = item{kind: kConn, path: e.Path, idx: e.Idx, name: where + " " + e.Conn.Name, label: label}
	}
	return items
}

type model struct {
	list      list.Model
	title     string
	data      comm.JsonData
	browse    []string
	searching bool // list holds searchItems instead of the current pod
	static    bool // plain chooser: no pods, no actions
	notice    string
	confirm   string // non-empty: waiting for y/n to delete this
	confirmIt item
	result    Result
}

func (m model) Init() tea.Cmd { return nil }

func (m *model) setTitle() {
	if len(m.browse) == 0 {
		m.list.Title = m.title
		return
	}
	m.list.Title = m.title + "  📂 /" + comm.JoinPath(m.browse)
}

// show replaces the list content with the current pod and puts the cursor on the given entry.
func (m *model) show(sel *item) {
	m.list.SetItems(levelItems(m.data, m.browse))
	m.list.ResetSelected()
	m.setTitle()
	if sel == nil {
		return
	}
	for pos, it := range m.list.Items() {
		if i := it.(item); i.kind == sel.kind && i.idx == sel.idx {
			m.list.Select(pos)
			return
		}
	}
}

func (m model) finish(a Action, it item) (tea.Model, tea.Cmd) {
	m.result = Result{Action: a, Browse: m.browse, Path: it.path, Idx: it.idx, IsPod: it.kind == kPod}
	return m, tea.Quit
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		m.notice = ""
		if m.confirm != "" {
			if k := msg.String(); k == "y" || k == "Y" {
				return m.finish(ActionDelete, m.confirmIt)
			}
			m.confirm = "" // anything else cancels
			return m, nil
		}
		state := m.list.FilterState()
		filtering := state == list.Filtering
		it, hasItem := m.list.SelectedItem().(item)

		switch msg.String() {
		case "ctrl+c", "ctrl+d":
			return m, tea.Quit
		case "q":
			if !filtering {
				return m, tea.Quit
			}
		case "/":
			if !m.static && state == list.Unfiltered {
				m.searching = true
				m.list.SetItems(searchItems(m.data))
				m.list.ResetSelected()
			}
		case "backspace":
			if !m.static && state == list.Unfiltered && len(m.browse) > 0 {
				return m.up()
			}
		case "enter":
			if filtering {
				break // let the list accept the search first
			}
			if !hasItem {
				return m, tea.Quit
			}
			switch it.kind {
			case kUp:
				return m.up()
			case kPod:
				m.browse = append(append([]string{}, m.browse...), it.name)
				m.show(nil)
				return m, nil
			}
			return m.finish(ActionConnect, it)
		case "a", "p":
			if m.static || filtering {
				break
			}
			a := ActionAdd
			if msg.String() == "p" {
				a = ActionNewPod
			}
			return m.finish(a, item{path: m.browse})
		case "e", "d", "m":
			if m.static || filtering {
				break
			}
			if !hasItem || it.kind == kUp || (msg.String() == "m" && it.kind == kPod) {
				return m, nil
			}
			switch msg.String() {
			case "e":
				return m.finish(ActionEdit, it)
			case "m":
				return m.finish(ActionMove, it)
			}
			m.confirmIt = it
			m.confirm = fmt.Sprintf("Delete '%s'?", it.name)
			if it.kind == kPod {
				_, pods, _ := m.data.Level(it.path)
				m.confirm = fmt.Sprintf("Delete pod '%s' and its %d items?", it.name, (*pods)[it.idx].Count())
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	if m.searching && m.list.FilterState() == list.Unfiltered {
		m.searching = false // search ended: back to the pod view
		m.show(nil)
	}
	return m, cmd
}

// up goes to the parent pod and keeps the cursor on the pod we came from.
func (m model) up() (tea.Model, tea.Cmd) {
	from := m.browse[len(m.browse)-1]
	m.browse = append([]string{}, m.browse[:len(m.browse)-1]...)
	var sel *item
	if _, pods, ok := m.data.Level(m.browse); ok {
		for i, p := range *pods {
			if p.Name == from {
				sel = &item{kind: kPod, idx: i}
			}
		}
	}
	m.show(sel)
	return m, nil
}

func (m model) View() string {
	if m.result.Action != ActionCancel {
		return ""
	}
	if m.confirm != "" {
		return "\n" + errStyle.Render("  "+m.confirm+" (y/N)") + "\n"
	}
	if len(m.list.Items()) == 0 && !m.static {
		return "\n" + titleStyle.Render(m.list.Title) + "\n\n" +
			itemStyle.Render("No items (Please add host)") + "\n\n" +
			helpStyle.Render("a add • p new pod • q quit") + "\n"
	}
	v := "\n" + m.list.View()
	if m.notice != "" {
		v += "\n" + errStyle.Render("  "+m.notice)
	}
	return v
}

func ago(ts int64) string {
	if ts == 0 {
		return ""
	}
	d := time.Since(time.Unix(ts, 0))
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

func newList(title string, items []list.Item) list.Model {
	l := list.New(items, itemDelegate{}, 20, listHeight)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	// the list's "filter" is a name search; label it that way
	l.FilterInput.Prompt = "Search: "
	l.KeyMap.Filter.SetHelp("/", "search")
	l.KeyMap.ClearFilter.SetHelp("esc", "clear search")
	l.KeyMap.AcceptWhileFiltering.SetHelp("enter", "apply search")
	l.KeyMap.CancelWhileFiltering.SetHelp("esc", "cancel search")
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	return l
}

func run(m model) (model, error) {
	ClearScreen()
	final, err := tea.NewProgram(m).Run()
	ClearScreen()
	if err != nil {
		return m, err
	}
	return final.(model), nil
}

// Browse shows the pod tree: enter opens a pod or connects, backspace goes up, '/' searches
// every pod, and a/e/d/m/p add, edit(rename), delete, move and create pods.
func Browse(title string, data comm.JsonData, st State) (Result, error) {
	if _, _, ok := data.Level(st.Path); !ok {
		st.Path = nil
	}
	l := newList(title, nil)
	extra := []key.Binding{
		key.NewBinding(key.WithKeys("backspace"), key.WithHelp("backspace", "up")),
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "move")),
		key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "new pod")),
	}
	l.AdditionalShortHelpKeys = func() []key.Binding { return extra }
	l.AdditionalFullHelpKeys = func() []key.Binding { return extra }

	m := model{list: l, title: title, data: data, browse: st.Path, notice: st.Notice}
	var sel *item
	if st.HasSel {
		k := kConn
		if st.SelPod {
			k = kPod
		}
		sel = &item{kind: k, idx: st.SelIdx}
	}
	m.show(sel)

	fm, err := run(m)
	if err != nil {
		return Result{}, err
	}
	return fm.result, nil
}

// choose is a plain single-choice list; it returns the idx of the chosen item.
func choose(title string, items []list.Item) (int, bool, error) {
	fm, err := run(model{list: newList(title, items), title: title, static: true})
	if err != nil {
		return 0, false, err
	}
	return fm.result.Idx, fm.result.Action == ActionConnect, nil
}

// Pick shows a searchable list of connections (with their pod) and returns the index into entries.
func Pick(title string, entries []comm.Entry) (idx int, ok bool, err error) {
	order := make([]int, len(entries))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return entries[order[a]].Conn.LastUsed > entries[order[b]].Conn.LastUsed })
	items := make([]list.Item, len(entries))
	for pos, i := range order {
		e := entries[i]
		label := connLabel(e.Conn)
		if where := comm.JoinPath(e.Path); where != "" {
			label = "📁 " + where + " › " + label
		}
		items[pos] = item{kind: kConn, idx: i, name: comm.JoinPath(e.Path) + " " + e.Conn.Name, label: label}
	}
	return choose(title, items)
}

// PickPod lets the user choose one of paths (nil is the top level); it returns the index into paths.
func PickPod(title string, paths [][]string) (idx int, ok bool, err error) {
	items := make([]list.Item, len(paths))
	for i, p := range paths {
		name := "/" + comm.JoinPath(p)
		items[i] = item{kind: kConn, idx: i, name: name, label: "📁 " + name}
	}
	return choose(title, items)
}

type promptModel struct {
	title string
	in    textinput.Model
	ok    bool
}

func (m promptModel) Init() tea.Cmd { return textinput.Blink }

func (m promptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, isKey := msg.(tea.KeyMsg); isKey {
		switch k.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			m.ok = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.in, cmd = m.in.Update(msg)
	return m, cmd
}

func (m promptModel) View() string {
	return "\n" + titleStyle.Render(m.title) + "\n\n  " + m.in.View() +
		"\n\n" + helpStyle.Render("enter confirm • esc cancel") + "\n"
}

// Prompt asks for one line of text; ok is false if cancelled.
func Prompt(title, initial string) (string, bool, error) {
	in := textinput.New()
	in.Prompt = "> "
	in.CharLimit = 128
	in.SetValue(initial)
	in.Focus()
	ClearScreen()
	final, err := tea.NewProgram(promptModel{title: title, in: in}).Run()
	ClearScreen()
	if err != nil {
		return "", false, err
	}
	fm := final.(promptModel)
	return strings.TrimSpace(fm.in.Value()), fm.ok, nil
}

// ClearScreen clears the terminal (before/after a TUI and around the ssh session).
func ClearScreen() {
	out := termenv.DefaultOutput()
	out.ClearScreen()
	out.ShowCursor()
}
