package comm

import (
	"errors"
	"fmt"
	"strings"
)

// ParsePath splits "a/b/c" into pod names; blank parts are dropped, "" is the root.
func ParsePath(s string) []string {
	var out []string
	for _, p := range strings.Split(s, "/") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func JoinPath(p []string) string { return strings.Join(p, "/") }

func PathEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Level returns the conns and child pods of the pod at path (nil: root).
// The pointers are only valid until the tree is next modified.
func (d *JsonData) Level(path []string) (*[]Conn, *[]Pod, bool) {
	conns, pods := &d.JsonData, &d.Pods
	for _, name := range path {
		i := podIndex(*pods, name)
		if i < 0 {
			return nil, nil, false
		}
		conns, pods = &(*pods)[i].Conn, &(*pods)[i].Pods
	}
	return conns, pods, true
}

func podIndex(pods []Pod, name string) int {
	for i, p := range pods {
		if p.Name == name {
			return i
		}
	}
	return -1
}

// ensure walks path, creating missing pods, and returns that level.
func (d *JsonData) ensure(path []string) (*[]Conn, *[]Pod) {
	conns, pods := &d.JsonData, &d.Pods
	for _, name := range path {
		i := podIndex(*pods, name)
		if i < 0 {
			*pods = append(*pods, Pod{Name: name})
			i = len(*pods) - 1
		}
		conns, pods = &(*pods)[i].Conn, &(*pods)[i].Pods
	}
	return conns, pods
}

func hasName(conns []Conn, name string, skip int) bool {
	for i, c := range conns {
		if i != skip && c.Name == name {
			return true
		}
	}
	return false
}

// PutConn adds (idx < 0) c to pod `to`, edits it in place, or moves it from pod `from`
// when `to` differs. Missing pods in `to` are created. It returns c's new index in `to`.
// Names are unique per pod.
func (d *JsonData) PutConn(from []string, idx int, c Conn, to []string) (int, error) {
	if idx >= 0 {
		src, _, ok := d.Level(from)
		if !ok || idx >= len(*src) {
			return 0, errors.New("item not found")
		}
		if PathEqual(from, to) {
			if hasName(*src, c.Name, idx) {
				return 0, fmt.Errorf("name already exists: %s", c.Name)
			}
			(*src)[idx] = c
			return idx, nil
		}
	}
	if dst, _, ok := d.Level(to); ok && hasName(*dst, c.Name, -1) {
		return 0, fmt.Errorf("name already exists in pod: %s", c.Name)
	}
	if idx >= 0 {
		d.RemoveConn(from, idx)
	}
	dst, _ := d.ensure(to)
	*dst = append(*dst, c)
	return len(*dst) - 1, nil
}

// RemoveConn deletes conns[idx] of the pod at path.
func (d *JsonData) RemoveConn(path []string, idx int) {
	if conns, _, ok := d.Level(path); ok && idx >= 0 && idx < len(*conns) {
		*conns = append((*conns)[:idx], (*conns)[idx+1:]...)
	}
}

// AddPod creates the pod `sub` (may be nested, "a/b") under path.
func (d *JsonData) AddPod(path []string, sub string) error {
	names := ParsePath(sub)
	if len(names) == 0 {
		return errors.New("pod name must not be empty")
	}
	if _, pods, ok := d.Level(path); !ok {
		return errors.New("pod not found")
	} else if len(names) == 1 && podIndex(*pods, names[0]) >= 0 {
		return fmt.Errorf("pod already exists: %s", names[0])
	}
	d.ensure(append(append([]string{}, path...), names...))
	return nil
}

// RenamePod renames pods[idx] of the pod at path.
func (d *JsonData) RenamePod(path []string, idx int, name string) error {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return errors.New("pod name must not be empty")
	case strings.Contains(name, "/"):
		return errors.New("pod name must not contain '/'")
	}
	_, pods, ok := d.Level(path)
	if !ok || idx < 0 || idx >= len(*pods) {
		return errors.New("pod not found")
	}
	if i := podIndex(*pods, name); i >= 0 && i != idx {
		return fmt.Errorf("pod already exists: %s", name)
	}
	(*pods)[idx].Name = name
	return nil
}

// RemovePod deletes pods[idx] of the pod at path together with everything inside it.
func (d *JsonData) RemovePod(path []string, idx int) {
	if _, pods, ok := d.Level(path); ok && idx >= 0 && idx < len(*pods) {
		*pods = append((*pods)[:idx], (*pods)[idx+1:]...)
	}
}

// Count returns the number of connections in p, including child pods.
func (p Pod) Count() int {
	n := len(p.Conn)
	for _, c := range p.Pods {
		n += c.Count()
	}
	return n
}

// PodPaths lists every pod path depth-first; the root (nil) comes first.
func (d JsonData) PodPaths() [][]string {
	out := [][]string{nil}
	var walk func(prefix []string, pods []Pod)
	walk = func(prefix []string, pods []Pod) {
		for _, p := range pods {
			path := append(append([]string{}, prefix...), p.Name)
			out = append(out, path)
			walk(path, p.Pods)
		}
	}
	walk(nil, d.Pods)
	return out
}

// Entry is a connection with its location in the tree.
type Entry struct {
	Path []string
	Idx  int
	Conn Conn
}

// Flatten lists every connection (root first, then pods depth-first).
func (d JsonData) Flatten() []Entry {
	var out []Entry
	for _, path := range d.PodPaths() {
		conns, _, _ := d.Level(path)
		for i, c := range *conns {
			out = append(out, Entry{Path: path, Idx: i, Conn: c})
		}
	}
	return out
}

// EachConn calls fn for every connection in the tree.
func (d *JsonData) EachConn(fn func(*Conn)) {
	for _, path := range d.PodPaths() {
		conns, _, _ := d.Level(path)
		for i := range *conns {
			fn(&(*conns)[i])
		}
	}
}
