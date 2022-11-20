package cmd

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
)

const (
	ConfigWindowsFilePath = "\\.smg\\conn.json"
	ConfigUnixFilePath = "/.smg/conn.json"
)

type JsonData struct {
	JsonData []Conn `json:"conn"`
}

type Conn struct {
	Name      string `json:"name"`
	IP        string `json:"ip"`
	Port      int    `json:"port"`
	User      string `json:"user"`
	Cert_Type int    `json:"cert_type"`
	Cert      string `json:"cert"`
}

func CheckConnfile() string {

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	osInfo := runtime.GOOS

	filepath := homeDir

	switch osInfo {
	case "windows":
		filepath = homeDir + ConfigWindowsFilePath
	case "linux", "darwin":
		filepath = homeDir + ConfigUnixFilePath
	default:
		log.Printf("This OS type is not support! sorry!\n")
		os.Exit(-1)
	}

	log.Println("This Env OS:", osInfo)

	if _, err := os.Stat(filepath); err != nil {
		log.Printf("Connection file is not exist\n")
		os.Exit(-1)
	}

	return filepath
}

func LoadConnFile(filepath string) JsonData {
	jsonFile, _ := os.Open(filepath)
	log.Printf("Load Keyfile: %s\n", filepath)
	byteValue, _ := ioutil.ReadAll(jsonFile)

	var json_data JsonData

	json.Unmarshal(byteValue, &json_data)

	// check '~' and if exist then replace home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	for idx, dat := range json_data.JsonData {
        if(strings.ContainsAny(dat.Cert, "~")){
			json_data.JsonData[idx].Cert = strings.Replace(dat.Cert, "~", homeDir, 1)
		}
    }

	return json_data
}