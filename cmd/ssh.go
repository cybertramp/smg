package cmd

import (
	_ "bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	terminal "golang.org/x/term"
)

const (
	CertNoPassword    = 0
	CertPassword      = 1
	CertPublicKeyFile = 2

	DefaultTimeout = 5
)

type SSH struct {
	IP        string
	Port      int
	User      string
	Cert_Type int
	Cert      string
	session   *ssh.Session
	client    *ssh.Client
}

func (S *SSH) readPublicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

// Connect the SSH Server
func (S *SSH) Connect() error {
	var sshConfig *ssh.ClientConfig
	var auth []ssh.AuthMethod

	switch S.Cert_Type {
	case CertNoPassword:
		break
	case CertPassword:
		auth = []ssh.AuthMethod{
			ssh.Password(S.Cert),
		}
	case CertPublicKeyFile:
		// static to abs
		auth = []ssh.AuthMethod{
			S.readPublicKeyFile(S.Cert),
		}
	default:
		log.Println("Does not support cert type: ", S.Cert_Type)
		os.Exit(-1)
	}

	sshConfig = &ssh.ClientConfig{
		User: S.User,
		Auth: auth,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: time.Second * DefaultTimeout,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", S.IP, S.Port), sshConfig)
	if err != nil {
		log.Println("Dial Failed:", err)
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		log.Println(err)
		client.Close()
		return err
	}

	fd := int(os.Stdin.Fd())

	state, err := terminal.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("terminal make raw: %s", err)
	}
	defer terminal.Restore(fd, state)

	termW, termH, err := terminal.GetSize(fd)
	if err != nil {
		log.Fatalf("Can't get Terminal Width, Height: %s", err)
	}

	S.session = session
	S.client = client

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // input echo
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4k
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4k
	}

	term_type := os.Getenv("TERM")
	if term_type == "" {
		term_type = "xterm-256color"
	}

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	if err := session.RequestPty(term_type, termH, termW, modes); err != nil {
		log.Fatalf("request for pseudo terminal failed: %s", err)
	}

	if err := session.Shell(); err != nil {
		log.Fatalf("Shell is broken: %s", err)
	}

	// Control Signal
	sigChannel := make(chan os.Signal)
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			sig, ok := <-sigChannel
			if !ok {
				break
			}

			switch sig {
			case syscall.SIGINT:
				session.Signal(ssh.SIGINT)
			case syscall.SIGTERM:
				session.Signal(ssh.SIGTERM)
			}
		}
	}()

	return session.Wait()
}

// RunCmd to SSH Server
func (S *SSH) RunCmd(cmd string) {
	out, err := S.session.CombinedOutput(cmd)
	if err != nil {
		log.Fatalln("Error!", err)

	}
	log.Println(string(out))
}

// Session Close
func (S *SSH) Close() {
	if S.session.Close() != nil {
		log.Println("Failed session slose")
	}
	if S.client.Close() != nil {
		log.Println("Failed client slose")
	}
}

// Run SSH Client
func RunSSH(conn Conn) error {

	client := &SSH{
		IP:        conn.IP,
		Port:      conn.Port,
		User:      conn.User,
		Cert_Type: conn.Cert_Type,
		Cert:      conn.Cert,
	}

	err := client.Connect()
	if err != nil {
		return err
	}
	client.Close()

	return err
}
