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
)

const ProgramVersion = "0.1.0"

const help = `smg v%s
Simple SSH Manager with Go
Program repository: github.com/cybertramp/smg

    (none): Select a connection and connect (/ to filter)
    help   : Manual for this program
    init   : Make default connection config file
    add    : Add new connection item
    edit   : Edit connection item
    delete : Delete connection item
    version: Show version
`

func load() (string, comm.JsonData, error) {
	path, err := config.ExistingPath()
	if err != nil {
		return "", comm.JsonData{}, err
	}
	data, err := config.Load(path)
	return path, data, err
}

func connect() error {
	path, data, err := load()
	if err != nil {
		return err
	}
	if len(data.JsonData) == 0 {
		return fmt.Errorf("no connection data: run 'smg add' (%s)", path)
	}

	idx, ok, err := tuimain.Pick("Choose where to connect with ssh? 🚲", data.JsonData)
	if err != nil || !ok {
		return err
	}

	tuimain.ClearScreen()
	err = ssh.Run(data.JsonData[idx])
	if err == nil {
		fmt.Println("Session is closed! Bye ✋")
	}
	return err
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

func initConfig() error {
	path, created, err := config.Init()
	if err != nil {
		return err
	}
	if created {
		fmt.Println("Create config file:", path)
	} else {
		fmt.Println("Already exist connection config file:", path)
	}
	return nil
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
	case "init":
		return initConfig()
	case "add":
		return add()
	case "edit":
		return edit()
	case "delete":
		return del()
	default:
		return fmt.Errorf("unsupported command: %s (see 'smg help')", args[0])
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
