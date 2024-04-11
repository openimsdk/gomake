package mageutil

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"runtime"
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
		fmt.Printf("error reading YAML file: %v", err)
		os.Exit(1)
	}

	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		fmt.Printf("error unmarshalling YAML: %v", err)
		os.Exit(1)
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
}
