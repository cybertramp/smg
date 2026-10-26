package ssh

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
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

const (
	DefaultTimeout    = 10 * time.Second // TCP connect + SSH handshake
	keepAliveInterval = 30 * time.Second
)

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

// authMethods returns the auth methods for c. pause is called before any prompt
// that may block during the handshake (it disarms the handshake deadline).
func authMethods(c comm.Conn, pause func()) ([]ssh.AuthMethod, error) {
	switch c.CertType {
	case comm.CertAgent:
		m, err := agentAuth()
		return []ssh.AuthMethod{m}, err
	case comm.CertPassword:
		pass := c.Cert
		if pass == "" {
			var err error
			if pass, err = readPassword(fmt.Sprintf("%s@%s password: ", c.User, c.IP)); err != nil {
				return nil, err
			}
		}
		// Many servers disable "password" and only offer keyboard-interactive.
		used := false
		ki := ssh.KeyboardInteractive(func(_, _ string, questions []string, echos []bool) ([]string, error) {
			answers := make([]string, len(questions))
			for i, q := range questions {
				if !used && i == 0 {
					used = true
					answers[i] = pass
					continue
				}
				pause()
				fmt.Fprint(os.Stderr, q)
				if echos[i] {
					line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
					answers[i] = strings.TrimRight(line, "\r\n")
					continue
				}
				p, err := readPassword("")
				if err != nil {
					return nil, err
				}
				answers[i] = p
			}
			return answers, nil
		})
		return []ssh.AuthMethod{ssh.Password(pass), ki}, nil
	case comm.CertKeyFile:
		m, err := keyAuth(c.Cert)
		return []ssh.AuthMethod{m}, err
	}
	return nil, fmt.Errorf("does not support cert type: %d", c.CertType)
}

// hostKeyCallback verifies the server key against ~/.ssh/known_hosts.
// Unknown hosts are added after user confirmation; changed keys are rejected.
// algorithms are the host key types already known for addr (so a server offering
// another key type is not mistaken for a changed key).
func hostKeyCallback(addr string, remote net.Addr, pause func()) (cb ssh.HostKeyCallback, algorithms []string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}
	sshDir := filepath.Join(home, ".ssh")
	khPath := filepath.Join(sshDir, "known_hosts")

	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return nil, nil, err
	}
	f, err := os.OpenFile(khPath, os.O_CREATE|os.O_RDONLY, 0600)
	if err != nil {
		return nil, nil, err
	}
	_ = f.Close()

	check, err := knownhosts.New(khPath)
	if err != nil {
		return nil, nil, err
	}

	cb = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := check(hostname, remote, key)
		var keyErr *knownhosts.KeyError
		if !errors.As(err, &keyErr) {
			return err
		}
		if len(keyErr.Want) > 0 {
			return fmt.Errorf("HOST KEY CHANGED for %s (possible MITM): %w", hostname, err)
		}

		pause()
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
	}
	return cb, knownAlgorithms(check, addr, remote), nil
}

// knownAlgorithms lists the host key algorithms recorded for addr in known_hosts
// by probing with a throwaway key (KeyError.Want holds the recorded keys).
func knownAlgorithms(check ssh.HostKeyCallback, addr string, remote net.Addr) []string {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil
	}
	probe, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil
	}
	var keyErr *knownhosts.KeyError
	if !errors.As(check(addr, remote, probe), &keyErr) {
		return nil
	}
	var algs []string
	for _, w := range keyErr.Want {
		switch t := w.Key.Type(); t {
		case ssh.KeyAlgoRSA:
			algs = append(algs, ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSA)
		default:
			algs = append(algs, t)
		}
	}
	return algs
}

// dial connects to addr; timeout covers both TCP connect and SSH handshake.
// pause (passed to prompts) disarms the deadline so user input is not cut off.
func dial(c comm.Conn) (*ssh.Client, error) {
	addr := net.JoinHostPort(c.IP, fmt.Sprint(c.Port))
	conn, err := net.DialTimeout("tcp", addr, DefaultTimeout)
	if err != nil {
		return nil, err
	}
	pause := func() { _ = conn.SetDeadline(time.Time{}) }

	auth, err := authMethods(c, pause)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	hostKeyCb, algs, err := hostKeyCallback(addr, conn.RemoteAddr(), pause)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("known_hosts: %w", err)
	}

	_ = conn.SetDeadline(time.Now().Add(DefaultTimeout))
	sc, chans, reqs, err := ssh.NewClientConn(conn, addr, &ssh.ClientConfig{
		User:              c.User,
		Auth:              auth,
		HostKeyCallback:   hostKeyCb,
		HostKeyAlgorithms: algs,
		Timeout:           DefaultTimeout,
	})
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	return ssh.NewClient(sc, chans, reqs), nil
}

// keepAlive probes the server until done is closed; a dead link closes client.
func keepAlive(client *ssh.Client, done <-chan struct{}) {
	t := time.NewTicker(keepAliveInterval)
	defer t.Stop()
	for {
		select {
		case <-done:
			return
		case <-t.C:
			if _, _, err := client.SendRequest("keepalive@openssh.com", true, nil); err != nil {
				_ = client.Close()
				return
			}
		}
	}
}

// Run opens an interactive shell on c. A remote non-zero exit is returned as *ssh.ExitError (see ExitCode).
func Run(c comm.Conn) error {
	client, err := dial(c)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	done := make(chan struct{})
	defer close(done)
	go keepAlive(client, done)

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()

	fd := int(os.Stdin.Fd())
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

	// Raw mode only after setup so errors above print normally.
	state, err := terminal.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("terminal make raw: %w", err)
	}
	defer func() { _ = terminal.Restore(fd, state) }()

	// Best-effort: UTF-8 locale (needs server AcceptEnv).
	for _, k := range []string{"LANG", "LC_ALL"} {
		if v := os.Getenv(k); v != "" {
			_ = session.Setenv(k, v)
		}
	}

	if err := session.Shell(); err != nil {
		return fmt.Errorf("shell start failed: %w", err)
	}

	defer watchSignals(fd, session)()

	return session.Wait()
}
