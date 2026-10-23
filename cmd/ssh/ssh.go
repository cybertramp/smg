package ssh

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"smg/cmd/comm"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
	terminal "golang.org/x/term"
)

const DefaultTimeout = 3 * time.Second

// ExitCode returns the remote exit status if err is a remote non-zero exit.
func ExitCode(err error) (int, bool) {
	var ee *ssh.ExitError
	if errors.As(err, &ee) {
		return ee.ExitStatus(), true
	}
	return 0, false
}

func readPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	b, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	return string(b), err
}

func keyAuth(file string) (ssh.AuthMethod, error) {
	buf, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read key file (%s): %w", file, err)
	}

	key, err := ssh.ParsePrivateKey(buf)
	var missing *ssh.PassphraseMissingError
	if errors.As(err, &missing) {
		pass, perr := readPassword(fmt.Sprintf("Passphrase for %s: ", file))
		if perr != nil {
			return nil, perr
		}
		key, err = ssh.ParsePrivateKeyWithPassphrase(buf, []byte(pass))
	}
	if err != nil {
		return nil, fmt.Errorf("parse key file (%s): %w", file, err)
	}
	return ssh.PublicKeys(key), nil
}

func agentAuth() (ssh.AuthMethod, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, errors.New("SSH_AUTH_SOCK is not set (ssh-agent not running)")
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, fmt.Errorf("connect ssh-agent: %w", err)
	}
	return ssh.PublicKeysCallback(agent.NewClient(conn).Signers), nil
}

func authMethod(c comm.Conn) (ssh.AuthMethod, error) {
	switch c.CertType {
	case comm.CertAgent:
		return agentAuth()
	case comm.CertPassword:
		pass := c.Cert
		if pass == "" {
			var err error
			if pass, err = readPassword(fmt.Sprintf("%s@%s password: ", c.User, c.IP)); err != nil {
				return nil, err
			}
		}
		return ssh.Password(pass), nil
	case comm.CertKeyFile:
		return keyAuth(c.Cert)
	}
	return nil, fmt.Errorf("does not support cert type: %d", c.CertType)
}

// hostKeyCallback verifies the server key against ~/.ssh/known_hosts.
// Unknown hosts are added after user confirmation; changed keys are rejected.
func hostKeyCallback() (ssh.HostKeyCallback, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	sshDir := filepath.Join(home, ".ssh")
	khPath := filepath.Join(sshDir, "known_hosts")

	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(khPath, os.O_CREATE|os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}
	_ = f.Close()

	check, err := knownhosts.New(khPath)
	if err != nil {
		return nil, err
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := check(hostname, remote, key)
		var keyErr *knownhosts.KeyError
		if !errors.As(err, &keyErr) {
			return err
		}
		if len(keyErr.Want) > 0 {
			return fmt.Errorf("HOST KEY CHANGED for %s (possible MITM): %w", hostname, err)
		}

		fmt.Printf("Unknown host %s\n%s %s\nAdd to known_hosts? (yes/no): ",
			hostname, key.Type(), ssh.FingerprintSHA256(key))
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.TrimSpace(strings.ToLower(answer)) != "yes" {
			return errors.New("host key rejected by user")
		}

		kh, err := os.OpenFile(khPath, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(kh, knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key))
		if cerr := kh.Close(); err == nil {
			err = cerr
		}
		return err
	}, nil
}

// Run opens an interactive shell on c. A remote non-zero exit is returned as *ssh.ExitError (see ExitCode).
func Run(c comm.Conn) error {
	auth, err := authMethod(c)
	if err != nil {
		return err
	}
	hostKeyCb, err := hostKeyCallback()
	if err != nil {
		return fmt.Errorf("known_hosts: %w", err)
	}

	client, err := ssh.Dial("tcp", net.JoinHostPort(c.IP, fmt.Sprint(c.Port)), &ssh.ClientConfig{
		User:            c.User,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: hostKeyCb,
		Timeout:         DefaultTimeout,
	})
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()

	fd := int(os.Stdin.Fd())
	state, err := terminal.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("terminal make raw: %w", err)
	}
	defer func() { _ = terminal.Restore(fd, state) }()

	w, h, err := terminal.GetSize(fd)
	if err != nil {
		return fmt.Errorf("can't get terminal size: %w", err)
	}

	termType := os.Getenv("TERM")
	if termType == "" {
		termType = "xterm-256color"
	}

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty(termType, h, w, modes); err != nil {
		return fmt.Errorf("request for pseudo terminal failed: %w", err)
	}
	if err := session.Shell(); err != nil {
		return fmt.Errorf("shell start failed: %w", err)
	}

	defer watchSignals(fd, session)()

	return session.Wait()
}
