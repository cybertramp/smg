//go:build !windows

package ssh

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh"
	terminal "golang.org/x/term"
)

// watchSignals forwards terminal resize and SIGTERM to the session.
// Ctrl+C is passed through as a byte in raw mode, so SIGINT is not handled.
func watchSignals(fd int, session *ssh.Session) (stop func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGWINCH)
	go func() {
		for sig := range ch {
			switch sig {
			case syscall.SIGTERM:
				_ = session.Signal(ssh.SIGTERM)
			case syscall.SIGWINCH:
				if w, h, err := terminal.GetSize(fd); err == nil {
					_ = session.WindowChange(h, w)
				}
			}
		}
	}()
	return func() {
		signal.Stop(ch)
		close(ch)
	}
}
