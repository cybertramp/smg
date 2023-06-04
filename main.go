package main

import (
	"fmt"
	"log"
	"os"
	"smg/cmd/comm"
	"smg/cmd/config"
	"smg/cmd/tui"
)

const ProgramVersion = "0.04"


func menu_help () {
    fmt.Println("smg v"+ProgramVersion)
    fmt.Println("Simple SSH Manager with Go")
    fmt.Println("Program repository: github.com/cybertramp/smg\n")
    fmt.Println("    help: Manual for this program")
    fmt.Println("    init: Make default connection config file")
    fmt.Println("    add : Add new connection item")
}

func menu_init () {
    config.MakeConnFile()
}

func menu_add() {
    config_file_path := config.GetConnFilePath()
    conn_json := config.LoadConnFile(config_file_path)
    tui.Run(config_file_path, conn_json, comm.TuiAddItem)
}

func menu_main() {
    config_file_path := config.GetConnFilePath()
    conn_json := config.LoadConnFile(config_file_path)
    tui.Run(config_file_path, conn_json, comm.TuiMain)
}

func main() {

    args := os.Args
    if len(args) > 1 {
        switch args[1]{
        case "help":
            menu_help()
        case "init":
            menu_init()
        case "add":
            menu_add()
        default:
            log.Fatalln("This command not support!")
        }
    }else{
        menu_main()
    }
}