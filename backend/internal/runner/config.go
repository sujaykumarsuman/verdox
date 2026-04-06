package runner

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type VerdoxConfig struct {
	Version   int           `yaml:"version"`
	FullClone bool          `yaml:"full_clone"`
	Suites    []SuiteConfig `yaml:"suites"`
}

type SuiteConfig struct {
	Name     string            `yaml:"name"`
	Type     string            `yaml:"type"`
	Image    string            `yaml:"image"`
	Command  string            `yaml:"command"`
	Timeout  int               `yaml:"timeout"`
	Env      map[string]string `yaml:"env"`
	Services []string          `yaml:"services"`
}

// LoadVerdoxConfig reads and parses a verdox.yaml config file from the workspace.
// Returns nil without error if the file doesn't exist.
func LoadVerdoxConfig(workDir, configPath string) (*VerdoxConfig, error) {
	path := configPath
	if path == "" {
		path = "verdox.yaml"
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(workDir, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg VerdoxConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// FindSuiteConfig looks up a suite by name in the verdox.yaml config.
func (c *VerdoxConfig) FindSuiteConfig(name string) *SuiteConfig {
	if c == nil {
		return nil
	}
	for i := range c.Suites {
		if c.Suites[i].Name == name {
			return &c.Suites[i]
		}
	}
	return nil
}
