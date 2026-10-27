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

    (none): Select a connection and connect (/ search all pods, a add, e edit, d delete, m move, p new pod)
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

// connectTo records the last-used time and opens the ssh session for the connection at pod/idx.
func connectTo(path string, data comm.JsonData, pod []string, idx int) error {
	conns, _, _ := data.Level(pod)
	(*conns)[idx].LastUsed = time.Now().Unix()
	if err := config.Save(path, data); err != nil {
		fmt.Fprintln(os.Stderr, "warning: can't save last-used time:", err)
	}

	tuimain.ClearScreen()
	err := ssh.Run((*conns)[idx])
	tuimain.ClearScreen()
	if err == nil {
		fmt.Println("Session is closed! Bye ✋")
	}
	return err
}

// connect is the main screen: browse pods, connect, and add/edit/delete/move entries.
func connect() error {
	path, data, err := load()
	if err != nil {
		return err
	}

	var st tuimain.State
	for {
		res, err := tuimain.Browse("Choose where to connect with ssh? 🚲", data, st)
		if err != nil {
			return err
		}
		st = tuimain.State{Path: res.Browse}

		save := func() {
			if err := config.Save(path, data); err != nil {
				st.Notice = "save failed: " + err.Error()
			}
		}
		fail := func(err error) { st.Notice = err.Error() }

		switch res.Action {
		case tuimain.ActionAdd:
			conn, dest, ok, err := tuiadd.Run(data, res.Browse, -1)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			idx, err := data.PutConn(nil, -1, conn, dest)
			if err != nil {
				fail(err)
				continue
			}
			save()
			st = tuimain.State{Path: dest, HasSel: true, SelIdx: idx}

		case tuimain.ActionEdit:
			if res.IsPod {
				_, pods, _ := data.Level(res.Path)
				name, ok, err := tuimain.Prompt("Rename pod", (*pods)[res.Idx].Name)
				if err != nil {
					return err
				}
				if !ok {
					continue
				}
				if err := data.RenamePod(res.Path, res.Idx, name); err != nil {
					fail(err)
					continue
				}
				save()
				st.HasSel, st.SelPod, st.SelIdx = true, true, res.Idx
				continue
			}
			conn, dest, ok, err := tuiadd.Run(data, res.Path, res.Idx)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			idx, err := data.PutConn(res.Path, res.Idx, conn, dest)
			if err != nil {
				fail(err)
				continue
			}
			save()
			if comm.PathEqual(dest, res.Browse) {
				st.HasSel, st.SelIdx = true, idx
			}

		case tuimain.ActionMove:
			conns, _, _ := data.Level(res.Path)
			conn := (*conns)[res.Idx]
			paths := data.PodPaths()
			n, ok, err := tuimain.PickPod(fmt.Sprintf("Move '%s' to", conn.Name), paths)
			if err != nil {
				return err
			}
			if !ok || comm.PathEqual(paths[n], res.Path) {
				continue
			}
			if _, err := data.PutConn(res.Path, res.Idx, conn, paths[n]); err != nil {
				fail(err)
				continue
			}
			save()

		case tuimain.ActionNewPod:
			name, ok, err := tuimain.Prompt("New pod name (a/b creates nested pods)", "")
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			if err := data.AddPod(res.Browse, name); err != nil {
				fail(err)
				continue
			}
			save()

		case tuimain.ActionDelete:
			if res.IsPod {
				data.RemovePod(res.Path, res.Idx)
			} else {
				data.RemoveConn(res.Path, res.Idx)
			}
			save()

		case tuimain.ActionConnect:
			return connectTo(path, data, res.Path, res.Idx)

		default:
			return nil
		}
	}
}

// connectByName connects to the entry called name in any pod: a unique exact match (case-insensitive)
// first, then a unique partial match; otherwise a list of the candidates opens.
func connectByName(name string) error {
	path, data, err := load()
	if err != nil {
		return err
	}

	var exact, partial []comm.Entry
	needle := strings.ToLower(name)
	for _, e := range data.Flatten() {
		switch n := strings.ToLower(e.Conn.Name); {
		case n == needle:
			exact = append(exact, e)
		case strings.Contains(n, needle):
			partial = append(partial, e)
		}
	}
	cands := exact
	if len(cands) == 0 {
		cands = partial
	}

	switch len(cands) {
	case 0:
		return fmt.Errorf("no connection matching %q (run 'smg' to list)", name)
	case 1:
		return connectTo(path, data, cands[0].Path, cands[0].Idx)
	}
	pick, ok, err := tuimain.Pick(fmt.Sprintf("Connections matching %q", name), cands)
	if err != nil || !ok {
		return err
	}
	return connectTo(path, data, cands[pick].Path, cands[pick].Idx)
}

func add() error {
	path, data, err := load()
	if err != nil {
		return err
	}
	conn, dest, ok, err := tuiadd.Run(data, nil, -1)
	if err != nil || !ok {
		return err
	}
	if _, err := data.PutConn(nil, -1, conn, dest); err != nil {
		return err
	}
	return config.Save(path, data)
}

// pickEntry lets the user choose any connection in the tree.
func pickEntry(title string, data comm.JsonData) (comm.Entry, bool, error) {
	entries := data.Flatten()
	if len(entries) == 0 {
		return comm.Entry{}, false, errors.New("no connection data")
	}
	i, ok, err := tuimain.Pick(title, entries)
	if err != nil || !ok {
		return comm.Entry{}, false, err
	}
	return entries[i], true, nil
}

func edit() error {
	path, data, err := load()
	if err != nil {
		return err
	}
	e, ok, err := pickEntry("Choose an item to edit", data)
	if err != nil || !ok {
		return err
	}
	conn, dest, ok, err := tuiadd.Run(data, e.Path, e.Idx)
	if err != nil || !ok {
		return err
	}
	if _, err := data.PutConn(e.Path, e.Idx, conn, dest); err != nil {
		return err
	}
	return config.Save(path, data)
}

func del() error {
	path, data, err := load()
	if err != nil {
		return err
	}
	e, ok, err := pickEntry("Choose an item to DELETE", data)
	if err != nil || !ok {
		return err
	}
	fmt.Printf("Delete '%s'? (yes/no): ", e.Conn.Name)
	var answer string
	_, _ = fmt.Scanln(&answer)
	if answer != "yes" {
		fmt.Println("Cancelled")
		return nil
	}
	data.RemoveConn(e.Path, e.Idx)
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
