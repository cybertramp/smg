package comm

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParsePath(t *testing.T) {
	if got := ParsePath(" /prod// web/ "); !reflect.DeepEqual(got, []string{"prod", "web"}) {
		t.Errorf("got %v", got)
	}
	if got := ParsePath(""); len(got) != 0 {
		t.Errorf("root must be empty, got %v", got)
	}
}

func TestPutConnAddMoveEdit(t *testing.T) {
	var d JsonData
	if _, err := d.PutConn(nil, -1, Conn{Name: "a"}, []string{"prod", "web"}); err != nil {
		t.Fatal(err)
	}
	if conns, _, ok := d.Level([]string{"prod", "web"}); !ok || len(*conns) != 1 {
		t.Fatalf("pod not created: %+v", d)
	}

	// edit in place
	if _, err := d.PutConn([]string{"prod", "web"}, 0, Conn{Name: "a2"}, []string{"prod", "web"}); err != nil {
		t.Fatal(err)
	}
	// move to root
	idx, err := d.PutConn([]string{"prod", "web"}, 0, Conn{Name: "a2"}, nil)
	if err != nil || idx != 0 || len(d.JsonData) != 1 {
		t.Fatalf("move: idx=%d err=%v data=%+v", idx, err, d)
	}
	if conns, _, _ := d.Level([]string{"prod", "web"}); len(*conns) != 0 {
		t.Error("source must be empty after move")
	}
}

func TestPutConnDuplicatePerPod(t *testing.T) {
	d := JsonData{JsonData: []Conn{{Name: "a"}}}
	if _, err := d.PutConn(nil, -1, Conn{Name: "a"}, nil); err == nil {
		t.Error("duplicate in same pod must fail")
	}
	if _, err := d.PutConn(nil, -1, Conn{Name: "a"}, []string{"x"}); err != nil {
		t.Errorf("same name in another pod is fine: %v", err)
	}
	// moving into a pod that already has the name must fail and leave the source intact
	if _, err := d.PutConn(nil, 0, Conn{Name: "a"}, []string{"x"}); err == nil {
		t.Error("move onto existing name must fail")
	}
	if len(d.JsonData) != 1 {
		t.Error("failed move must not remove the source")
	}
}

func TestPodOps(t *testing.T) {
	var d JsonData
	if err := d.AddPod(nil, "a/b"); err != nil {
		t.Fatal(err)
	}
	if err := d.AddPod(nil, "a"); err == nil {
		t.Error("existing pod must fail")
	}
	if err := d.RenamePod(nil, 0, "x/y"); err == nil {
		t.Error("slash in name must fail")
	}
	if err := d.RenamePod(nil, 0, "z"); err != nil || d.Pods[0].Name != "z" {
		t.Errorf("rename: %v %+v", err, d.Pods)
	}
	if _, err := d.PutConn(nil, -1, Conn{Name: "c"}, []string{"z", "b"}); err != nil {
		t.Fatal(err)
	}
	if d.Pods[0].Count() != 1 {
		t.Error("Count must include children")
	}
	if got := d.PodPaths(); len(got) != 3 {
		t.Errorf("paths = %v", got)
	}
	if got := d.Flatten(); len(got) != 1 || got[0].Conn.Name != "c" {
		t.Errorf("flatten = %+v", got)
	}
	d.RemovePod(nil, 0)
	if len(d.Pods) != 0 {
		t.Error("pod not removed")
	}
}

func TestLegacyJSONLoads(t *testing.T) {
	var d JsonData
	if err := json.Unmarshal([]byte(`{"conn":[{"name":"a"}]}`), &d); err != nil || len(d.JsonData) != 1 || d.Pods != nil {
		t.Fatalf("legacy file: %v %+v", err, d)
	}
	b, _ := json.Marshal(d)
	if string(b) != `{"conn":[{"name":"a","ip":"","port":0,"user":"","cert_type":0,"cert":""}]}` {
		t.Errorf("no pods key expected when empty: %s", b)
	}
}
