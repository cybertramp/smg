//go:build windows

package ssh

import "golang.org/x/crypto/ssh"

func watchSignals(fd int, session *ssh.Session) (stop func()) {
	return func() {}
}
