package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const defaultFoo = "Bar"

// Config captures runtime configuration loaded from YAML.
type Config struct {
	Foo string `yaml:"foo"`
}

// Load parses a YAML config file from disk.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Foo == "" {
		cfg.Foo = defaultFoo
	}

	return &cfg, nil
}
