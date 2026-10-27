package tuimain

import (
	"smg/cmd/comm"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAgo(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		ts   int64
		want string
	}{
		{0, ""},
		{now.Unix(), "just now"},
		{now.Add(-5 * time.Minute).Unix(), "5m ago"},
		{now.Add(-3 * time.Hour).Unix(), "3h ago"},
		{now.Add(-50 * time.Hour).Unix(), "2d ago"},
	} {
		if got := ago(tc.ts); got != tc.want {
			t.Errorf("ago(%d) = %q, want %q", tc.ts, got, tc.want)
		}
	}
}

func press(m model, k tea.KeyMsg) model {
	nm, _ := m.Update(k)
	return nm.(model)
}

func runes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func testModel(t *testing.T) model {
	t.Helper()
	d := comm.JsonData{JsonData: []comm.Conn{{Name: "top", IP: "h", Port: 22, User: "u"}}}
	for _, c := range []struct {
		name string
		pod  []string
	}{{"web1", []string{"prod", "web"}}, {"db1", []string{"prod"}}} {
		if _, err := d.PutConn(nil, -1, comm.Conn{Name: c.name, IP: "h", Port: 22, User: "u"}, c.pod); err != nil {
			t.Fatal(err)
		}
	}
	m := model{list: newList("t", nil), title: "t", data: d}
	m.list.SetSize(80, 20)
	m.show(nil)
	return m
}

func TestNavigateAndSearch(t *testing.T) {
	m := testModel(t)
	if n := len(m.list.Items()); n != 2 { // pod prod + conn top
		t.Fatalf("root items = %d, want 2", n)
	}

	m = press(m, tea.KeyMsg{Type: tea.KeyEnter}) // cursor on pod "prod" -> open
	if comm.JoinPath(m.browse) != "prod" {
		t.Fatalf("browse = %v", m.browse)
	}
	if n := len(m.list.Items()); n != 3 { // .., web/, db1
		t.Fatalf("prod items = %d, want 3", n)
	}

	m = press(m, tea.KeyMsg{Type: tea.KeyBackspace})
	if len(m.browse) != 0 {
		t.Fatalf("backspace must go up, browse = %v", m.browse)
	}

	m = press(m, runes("/")) // search spans every pod
	if !m.searching || len(m.list.Items()) != 3 {
		t.Fatalf("search items = %d searching=%v, want all 3 conns", len(m.list.Items()), m.searching)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.searching || len(m.list.Items()) != 2 {
		t.Fatalf("esc must restore pod view, items = %d", len(m.list.Items()))
	}
}

func TestPodActions(t *testing.T) {
	m := testModel(t) // cursor on pod "prod"

	m = press(m, runes("d"))
	if m.confirm == "" || !strings.Contains(m.confirm, "2 items") {
		t.Fatalf("confirm = %q", m.confirm)
	}
	m = press(m, runes("n"))
	if m.confirm != "" || m.result.Action != ActionCancel {
		t.Fatal("any other key must cancel the delete")
	}

	m = press(m, runes("m"))
	if m.result.Action != ActionCancel {
		t.Error("move is for connections only")
	}

	m = press(m, runes("d"))
	m = press(m, runes("y"))
	if r := m.result; r.Action != ActionDelete || !r.IsPod || r.Idx != 0 {
		t.Errorf("result = %+v", r)
	}
}

func TestConnectResult(t *testing.T) {
	m := testModel(t)
	m = press(m, tea.KeyMsg{Type: tea.KeyDown}) // top
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if r := m.result; r.Action != ActionConnect || r.IsPod || len(r.Path) != 0 || r.Idx != 0 {
		t.Errorf("result = %+v", r)
	}
}
