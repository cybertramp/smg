package main

import (
	"smg/cmd"
)

func main() {

    conn_json := cmd.LoadConnFile(cmd.CheckConnfile())

    cmd.TuiRun(conn_json)
}