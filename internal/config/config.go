package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const defaultHookTimeoutSeconds = 5

// Config captures runtime configuration loaded from YAML.
type Config struct {
	Hooks []HookConfig `yaml:"hooks"`
}

// HookConfig controls the optional SOAPAction/XPath bridge.
type HookConfig struct {
	SOAPAction     string `yaml:"soapAction"`
	XPath          string `yaml:"xpath"`
	Endpoint       string `yaml:"endpoint"`
	TimeoutSeconds int    `yaml:"timeoutSeconds"`
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

	cfg.Hooks, err = sanitizeHooks(cfg.Hooks)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func sanitizeHooks(in []HookConfig) ([]HookConfig, error) {
	var hooks []HookConfig
	for _, h := range in {
		if h.SOAPAction == "" && h.XPath == "" && h.Endpoint == "" {
			continue
		}
		if h.SOAPAction == "" || h.XPath == "" || h.Endpoint == "" {
			return nil, fmt.Errorf("hook config: soapAction, xpath, and endpoint must all be set or all be empty")
		}
		if h.TimeoutSeconds <= 0 {
			h.TimeoutSeconds = defaultHookTimeoutSeconds
		}
		hooks = append(hooks, h)
	}
	return hooks, nil
}
