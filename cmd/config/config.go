package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"path/filepath"
	"smg/cmd/comm"
)

const (
	ConfigWindowsFilePath = "\\.smg\\conn.json"
	ConfigUnixFilePath = "/.smg/conn.json"
)

func GetConnFilePath() string{

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

    osInfo := runtime.GOOS

	filepath := ""

    switch osInfo {
	case "windows":
		filepath =  homeDir + ConfigWindowsFilePath
	case "linux", "darwin":
		filepath = homeDir + ConfigUnixFilePath
	default:
		log.Printf("This OS type is not support! sorry!\n")
		os.Exit(-1)
	}

	if _, err := os.Stat(filepath); err != nil {
		log.Fatalf("Connection info file is not exist(%s)\n", filepath)
		log.Fatalln("Please run 'smg init' 🥏")
		os.Exit(-1)
	}

	return filepath
}

func LoadConnFile(filepath string) comm.JsonData {
	jsonFile, _ := os.Open(filepath)
	log.Printf("Load Config file: %s\n", filepath)
	byteValue, _ := ioutil.ReadAll(jsonFile)

	var json_data comm.JsonData

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

func MakeConnFile() string{
	filepath_str := ""
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	osInfo := runtime.GOOS

	switch osInfo {
	case "windows":
		filepath_str =  homeDir + ConfigWindowsFilePath
	case "linux", "darwin":
		filepath_str = homeDir + ConfigUnixFilePath
	default:
		log.Printf("This OS type is not support! sorry!\n")
		os.Exit(-1)
	}

	if _, err := os.Stat(filepath_str); err == nil {
		log.Println("Already exist connection config file!")
		log.Println("-> filepath:", filepath_str)
		return filepath_str
	}

	dir_name_str := filepath.Join(homeDir, ".smg")

	if _, err := os.Stat(dir_name_str); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(dir_name_str, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}

	data := comm.JsonData{[]comm.Conn{}}
	json_data, _ := json.MarshalIndent(data,"", "    ")

	err = ioutil.WriteFile(filepath_str, json_data, os.FileMode(0644))
	if err != nil {
		log.Println(err)
	}

	log.Println("Create config file: "+ filepath_str)

	return filepath_str
}