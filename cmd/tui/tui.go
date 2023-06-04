package tui

import (
	"smg/cmd/comm"
	"smg/cmd/tui/tui_add_item"
	"smg/cmd/tui/tui_main"
)


func Run(config_file_path string, config comm.JsonData, tuiType int){
	switch tuiType{
	case comm.TuiMain:
		tui_main.TuiRun(config_file_path, config)
	case comm.TuiAddItem:
		tui_add_item.TuiRun(config_file_path, config)
	}
}