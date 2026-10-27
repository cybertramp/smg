package tuiadd

import (
	"smg/cmd/comm"
	"testing"
)

func fill(m *model, vals [nFields]string) {
	for i, v := range vals {
		m.inputs[i].SetValue(v)
	}
}

func TestBuild(t *testing.T) {
	m := newModel(comm.JsonData{}, nil, -1)
	fill(&m, [nFields]string{"a", "", "host", "", "u", "0", ""})

	c, err := m.build()
	if err != nil {
		t.Fatal(err)
	}
	if c.Port != comm.DefaultPort || c.Name != "a" || c.CertType != comm.CertAgent {
		t.Errorf("unexpected conn: %+v", c)
	}
}

func TestBuildErrors(t *testing.T) {
	for _, vals := range [][nFields]string{
		{"a", "", "h", "abc", "u", "0", ""}, // bad port
		{"a", "", "h", "22", "u", "", ""},   // missing cert type
		{"a", "", "h", "22", "u", "2", ""},  // keyfile without path
		{"", "", "h", "22", "u", "0", ""},   // empty name
	} {
		m := newModel(comm.JsonData{}, nil, -1)
		fill(&m, vals)
		if _, err := m.build(); err == nil {
			t.Errorf("expected error for %v", vals)
		}
	}
}

func TestEditPrefillAndDuplicate(t *testing.T) {
	conns := []comm.Conn{
		{Name: "a", IP: "h", Port: 2222, User: "u", CertType: comm.CertAgent},
		{Name: "b", IP: "h", Port: 22, User: "u", CertType: comm.CertAgent},
	}
	m := newModel(comm.JsonData{JsonData: conns}, nil, 0)
	if m.inputs[fPort].Value() != "2222" {
		t.Errorf("port not prefilled: %q", m.inputs[fPort].Value())
	}
	if _, err := m.build(); err != nil {
		t.Errorf("editing self must pass: %v", err)
	}
	m.inputs[fName].SetValue("b")
	if _, err := m.build(); err == nil {
		t.Error("renaming to existing name must fail")
	}
}

func TestPodField(t *testing.T) {
	d := comm.JsonData{JsonData: []comm.Conn{{Name: "a", IP: "h", Port: 22, User: "u"}}}
	if _, err := d.PutConn(nil, -1, comm.Conn{Name: "b", IP: "h", Port: 22, User: "u"}, []string{"prod"}); err != nil {
		t.Fatal(err)
	}

	// same name is fine in another pod, but not in the same one
	m := newModel(d, nil, -1)
	fill(&m, [nFields]string{"b", "", "h", "", "u", "0", ""})
	if _, err := m.build(); err != nil {
		t.Errorf("name free at top level: %v", err)
	}
	fill(&m, [nFields]string{"b", " /prod/ ", "h", "", "u", "0", ""})
	if _, err := m.build(); err == nil {
		t.Error("duplicate inside pod must fail")
	}
	if got := comm.JoinPath(m.destPod()); got != "prod" {
		t.Errorf("destPod = %q", got)
	}

	// editing prod/b: prefilled pod, self is not a duplicate
	m = newModel(d, []string{"prod"}, 0)
	if m.inputs[fPod].Value() != "prod" {
		t.Errorf("pod not prefilled: %q", m.inputs[fPod].Value())
	}
	if _, err := m.build(); err != nil {
		t.Errorf("editing self: %v", err)
	}
}
