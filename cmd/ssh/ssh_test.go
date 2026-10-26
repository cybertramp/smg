package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"smg/cmd/comm"
	"testing"

	gossh "golang.org/x/crypto/ssh"
)

func TestKeyAuthUnencrypted(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	blk, err := gossh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "id")
	if err := os.WriteFile(path, pem.EncodeToMemory(blk), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := authMethods(comm.Conn{CertType: comm.CertKeyFile, Cert: path}, func() {}); err != nil {
		t.Errorf("valid key rejected: %v", err)
	}
}

func TestAuthMethodErrors(t *testing.T) {
	if _, err := authMethods(comm.Conn{CertType: comm.CertKeyFile, Cert: "/nonexistent"}, func() {}); err == nil {
		t.Error("missing key file must fail")
	}
	if _, err := authMethods(comm.Conn{CertType: 9}, func() {}); err == nil {
		t.Error("unknown cert type must fail")
	}
	t.Setenv("SSH_AUTH_SOCK", "")
	if _, err := authMethods(comm.Conn{CertType: comm.CertAgent}, func() {}); err == nil {
		t.Error("agent without SSH_AUTH_SOCK must fail")
	}
	if m, err := authMethods(comm.Conn{CertType: comm.CertPassword, Cert: "pw"}, func() {}); err != nil || len(m) != 2 {
		t.Errorf("stored password: %v", err)
	}
}
