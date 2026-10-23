package config

import (
	"os"
	"path/filepath"
	"smg/cmd/comm"
	"testing"
)

func TestExpandHome(t *testing.T) {
	cases := map[string]string{
		"~/.ssh/id_rsa": "/h/.ssh/id_rsa",
		"~":             "/h",
		"/a/~/b":        "/a/~/b",
		"~user/x":       "~user/x",
	}
	for in, want := range cases {
		if got := expandHome(in, "/h"); got != want {
			t.Errorf("expandHome(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "conn.json")

	in := comm.JsonData{JsonData: []comm.Conn{
		{Name: "a", IP: "1.2.3.4", Port: 22, User: "u", CertType: comm.CertKeyFile, Cert: "~/k"},
	}}
	if err := Save(path, in); err != nil {
		t.Fatal(err)
	}

	if fi, _ := os.Stat(path); fi.Mode().Perm() != 0600 {
		t.Errorf("perm = %v, want 0600", fi.Mode().Perm())
	}

	out, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	if got := out.JsonData[0].Cert; got != home+"/k" {
		t.Errorf("cert = %q, want %q", got, home+"/k")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conn.json")
	if err := os.WriteFile(path, []byte("{bad"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestInit(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))

	p, created, err := Init()
	if err != nil || !created {
		t.Fatalf("Init: created=%v err=%v", created, err)
	}
	if _, created, _ := Init(); created {
		t.Error("second Init must not recreate")
	}
	if fi, _ := os.Stat(filepath.Dir(p)); fi.Mode().Perm() != 0700 {
		t.Errorf("dir perm = %v, want 0700", fi.Mode().Perm())
	}
}

func TestValidate(t *testing.T) {
	ok := comm.Conn{Name: "a", IP: "h", Port: 22, User: "u", CertType: comm.CertAgent}
	all := []comm.Conn{ok}

	tests := []struct {
		name string
		mod  func(c *comm.Conn)
		skip int
		bad  bool
	}{
		{"duplicate name", func(c *comm.Conn) {}, -1, true},
		{"edit self", func(c *comm.Conn) {}, 0, false},
		{"empty name", func(c *comm.Conn) { c.Name = "" }, -1, true},
		{"port 0", func(c *comm.Conn) { c.Name = "b"; c.Port = 0 }, -1, true},
		{"port 70000", func(c *comm.Conn) { c.Name = "b"; c.Port = 70000 }, -1, true},
		{"cert type 3", func(c *comm.Conn) { c.Name = "b"; c.CertType = 3 }, -1, true},
		{"keyfile w/o path", func(c *comm.Conn) { c.Name = "b"; c.CertType = comm.CertKeyFile }, -1, true},
		{"password blank ok", func(c *comm.Conn) { c.Name = "b"; c.CertType = comm.CertPassword }, -1, false},
	}
	for _, tc := range tests {
		c := ok
		tc.mod(&c)
		if err := Validate(c, all, tc.skip); (err != nil) != tc.bad {
			t.Errorf("%s: err=%v, bad=%v", tc.name, err, tc.bad)
		}
	}
}
