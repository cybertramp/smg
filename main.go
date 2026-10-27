package main

import (
	"errors"
	"fmt"
	"os"
	"smg/cmd/comm"
	"smg/cmd/config"
	"smg/cmd/ssh"
	"smg/cmd/tui/tuiadd"
	"smg/cmd/tui/tuimain"
	"strings"
	"time"
)

const ProgramVersion = "0.1.1"

const help = `smg v%s
Simple SSH Manager with Go
Program repository: github.com/cybertramp/smg

    (none): Select a connection and connect (/ search, a add, e edit, d delete)
    <name> : Connect directly to the item with this name (partial match ok)
    help   : Manual for this program
    add    : Add new connection item
    edit   : Edit connection item
    delete : Delete connection item
    version: Show version
`

// load reads the config file, creating an empty one on first run.
func load() (string, comm.JsonData, error) {
	path, created, err := config.Init()
	if err != nil {
		return "", comm.JsonData{}, err
	}
	if created {
		fmt.Fprintln(os.Stderr, "Create config file:", path)
	}
	data, err := config.Load(path)
	return path, data, err
}

// connectTo records the last-used time and opens the ssh session for data.JsonData[idx].
func connectTo(path string, data comm.JsonData, idx int) error {
	data.JsonData[idx].LastUsed = time.Now().Unix()
	if err := config.Save(path, data); err != nil {
		fmt.Fprintln(os.Stderr, "warning: can't save last-used time:", err)
	}

	tuimain.ClearScreen()
	err := ssh.Run(data.JsonData[idx])
	tuimain.ClearScreen()
	if err == nil {
		fmt.Println("Session is closed! Bye ✋")
	}
	return err
}

func connect() error {
	path, data, err := load()
	if err != nil {
		return err
	}

	sel := -1
	for {
		idx, act, err := tuimain.PickAction("Choose where to connect with ssh? 🚲", data.JsonData, sel)
		if err != nil {
			return err
		}
		switch act {
		case tuimain.ActionAdd:
			conn, ok, err := tuiadd.Run(data.JsonData, -1)
			if err != nil {
				return err
			}
			if ok {
				data.JsonData = append(data.JsonData, conn)
				if err := config.Save(path, data); err != nil {
					return err
				}
				sel = len(data.JsonData) - 1
			}
		case tuimain.ActionEdit:
			conn, ok, err := tuiadd.Run(data.JsonData, idx)
			if err != nil {
				return err
			}
			if ok {
				data.JsonData[idx] = conn
				if err := config.Save(path, data); err != nil {
					return err
				}
			}
			sel = idx
		case tuimain.ActionDelete:
			data.JsonData = append(data.JsonData[:idx], data.JsonData[idx+1:]...)
			if err := config.Save(path, data); err != nil {
				return err
			}
			sel = min(idx, len(data.JsonData)-1) // the entry that took its place
		case tuimain.ActionConnect:
			return connectTo(path, data, idx)
		default:
			return nil
		}
	}
}

// connectByName connects to the entry called name: exact match (case-insensitive) first,
// then a unique substring match; several matches open a list restricted to them.
func connectByName(name string) error {
	path, data, err := load()
	if err != nil {
		return err
	}

	var matches []int
	needle := strings.ToLower(name)
	for i, c := range data.JsonData {
		switch n := strings.ToLower(c.Name); {
		case n == needle:
			return connectTo(path, data, i)
		case strings.Contains(n, needle):
			matches = append(matches, i)
		}
	}

	switch len(matches) {
	case 0:
		return fmt.Errorf("no connection matching %q (run 'smg' to list)", name)
	case 1:
		return connectTo(path, data, matches[0])
	}

	subset := make([]comm.Conn, len(matches))
	for i, idx := range matches {
		subset[i] = data.JsonData[idx]
	}
	pick, ok, err := tuimain.Pick(fmt.Sprintf("Connections matching %q", name), subset)
	if err != nil || !ok {
		return err
	}
	return connectTo(path, data, matches[pick])
}

func add() error {
	path, data, err := load()
	if err != nil {
		return err
	}
	conn, ok, err := tuiadd.Run(data.JsonData, -1)
	if err != nil || !ok {
		return err
	}
	data.JsonData = append(data.JsonData, conn)
	return config.Save(path, data)
}

func edit() error {
	path, data, err := load()
	if err != nil {
		return err
	}
	if len(data.JsonData) == 0 {
		return errors.New("no connection data")
	}
	idx, ok, err := tuimain.Pick("Choose an item to edit", data.JsonData)
	if err != nil || !ok {
		return err
	}
	conn, ok, err := tuiadd.Run(data.JsonData, idx)
	if err != nil || !ok {
		return err
	}
	data.JsonData[idx] = conn
	return config.Save(path, data)
}

func del() error {
	path, data, err := load()
	if err != nil {
		return err
	}
	if len(data.JsonData) == 0 {
		return errors.New("no connection data")
	}
	idx, ok, err := tuimain.Pick("Choose an item to DELETE", data.JsonData)
	if err != nil || !ok {
		return err
	}
	name := data.JsonData[idx].Name
	fmt.Printf("Delete '%s'? (yes/no): ", name)
	var answer string
	_, _ = fmt.Scanln(&answer)
	if answer != "yes" {
		fmt.Println("Cancelled")
		return nil
	}
	data.JsonData = append(data.JsonData[:idx], data.JsonData[idx+1:]...)
	return config.Save(path, data)
}

func run(args []string) error {
	if len(args) == 0 {
		return connect()
	}
	switch args[0] {
	case "help", "-h", "--help":
		fmt.Printf(help, ProgramVersion)
	case "version", "-v", "--version":
		fmt.Println("smg v" + ProgramVersion)
	case "add":
		return add()
	case "edit":
		return edit()
	case "delete":
		return del()
	default:
		if len(args) > 1 {
			return fmt.Errorf("unsupported command: %s (see 'smg help')", args[0])
		}
		return connectByName(args[0])
	}
	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		if code, ok := ssh.ExitCode(err); ok {
			os.Exit(code) // remote shell exited non-zero
		}
		fmt.Fprintln(os.Stderr, "smg:", err)
		os.Exit(1)
	}
}
