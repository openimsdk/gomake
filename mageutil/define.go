package mageutil

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
)

var (
	serviceBinaries    map[string]int
	toolBinaries       []string
	MaxFileDescriptors int
)

type Config struct {
	ServiceBinaries    map[string]int `yaml:"serviceBinaries"`
	ToolBinaries       []string       `yaml:"toolBinaries"`
	MaxFileDescriptors int            `yaml:"maxFileDescriptors"`
}

func InitForSSC() {
	yamlFile, err := ioutil.ReadFile("start-config.yml")
	if err != nil {
		fmt.Println("3333333333333333333333: ", err.Error())
		pid := os.Getpid()

		pidStr := strconv.Itoa(pid)

		cmd := exec.Command("lsof", "-p", pidStr)
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			fmt.Println("Failed to run lsof:", err)
			return
		}
		fmt.Println("Open files:", out.String())

		log.Fatalf("error reading YAML file: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		fmt.Println("444444444444444444444")

		log.Fatalf("error unmarshalling YAML: %v", err)
	}

	adjustedBinaries := make(map[string]int)
	for binary, count := range config.ServiceBinaries {
		if runtime.GOOS == "windows" {
			binary += ".exe"
		}
		adjustedBinaries[binary] = count
	}

	serviceBinaries = adjustedBinaries
	toolBinaries = config.ToolBinaries
	MaxFileDescriptors = config.MaxFileDescriptors
	fmt.Println("555555555555555555")

}
