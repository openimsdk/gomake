package mageutil

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	StartConfigFile = "start-config.yml"
)

var (
	startConfig StartConfig
)

type StartConfig struct {
	Services           map[string]int `yaml:"services"`
	Tools              []string       `yaml:"tools"`
	MaxFileDescriptors int            `yaml:"maxFileDescriptors"`
}

func LoadStartConfig() error {
	yamlFile, err := os.ReadFile(filepath.Join(Paths.Root, StartConfigFile))
	if err != nil {
		return fmt.Errorf("error reading YAML file: %w", err)
	}

	var cfg StartConfig
	decoder := yaml.NewDecoder(bytes.NewReader(yamlFile))
	decoder.KnownFields(true)
	if err = decoder.Decode(&cfg); err != nil {
		PrintYellow("current start-config format not detected, trying legacy format")
		cfg, err = loadLegacyStartConfig(yamlFile)
		if err != nil {
			return fmt.Errorf("error unmarshalling YAML: %w", err)
		}
	}

	startConfig = cfg
	return nil
}

func loadLegacyStartConfig(yamlFile []byte) (StartConfig, error) {
	var cfg struct {
		Services           map[string]int `yaml:"serviceBinaries"`
		Tools              []string       `yaml:"toolBinaries"`
		MaxFileDescriptors int            `yaml:"maxFileDescriptors"`
	}
	decoder := yaml.NewDecoder(bytes.NewReader(yamlFile))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return StartConfig{}, fmt.Errorf("error unmarshalling legacy YAML: %w", err)
	}
	return StartConfig{
		Services:           cfg.Services,
		Tools:              cfg.Tools,
		MaxFileDescriptors: cfg.MaxFileDescriptors,
	}, nil
}
