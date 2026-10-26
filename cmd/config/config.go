package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"smg/cmd/comm"
	"strings"
)

// Path returns ~/.smg/conn.json.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".smg", "conn.json"), nil
}

// expandHome replaces a leading '~' with the home directory.
func expandHome(p, home string) string {
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		return home + p[1:]
	}
	return p
}

func Load(path string) (comm.JsonData, error) {
	var data comm.JsonData

	b, err := os.ReadFile(path)
	if err != nil {
		return data, err
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return data, fmt.Errorf("invalid config file (%s): %w", path, err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return data, err
	}
	for i, c := range data.JsonData {
		if c.CertType == comm.CertKeyFile {
			data.JsonData[i].Cert = expandHome(c.Cert, home)
		}
	}
	return data, nil
}

// Save writes atomically (temp file + rename) with 0600 permission.
func Save(path string, data comm.JsonData) error {
	b, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".conn-*.json")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op after successful rename

	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil { // CreateTemp already uses 0600
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Init creates an empty config file. It returns false if it already existed.
func Init() (path string, created bool, err error) {
	path, err = Path()
	if err != nil {
		return "", false, err
	}
	if _, err := os.Stat(path); err == nil {
		return path, false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", false, err
	}
	if err := Save(path, comm.JsonData{JsonData: []comm.Conn{}}); err != nil {
		return "", false, err
	}
	return path, true, nil
}

// Validate checks c against the other entries (skip is the index being edited, -1 for new).
func Validate(c comm.Conn, all []comm.Conn, skip int) error {
	switch {
	case c.Name == "":
		return fmt.Errorf("name must not be empty")
	case c.IP == "":
		return fmt.Errorf("ip must not be empty")
	case c.User == "":
		return fmt.Errorf("id must not be empty")
	case c.Port < 1 || c.Port > 65535:
		return fmt.Errorf("invalid port: %d", c.Port)
	case c.CertType < comm.CertAgent || c.CertType > comm.CertKeyFile:
		return fmt.Errorf("invalid cert type: %d (0: agent / 1: PW / 2: keyfile)", c.CertType)
	case c.CertType == comm.CertKeyFile && c.Cert == "":
		return fmt.Errorf("keyfile path must not be empty")
	}
	for i, o := range all {
		if i != skip && o.Name == c.Name {
			return fmt.Errorf("name already exists: %s", c.Name)
		}
	}
	return nil
}
