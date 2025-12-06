// Package config handles all configuration logic for hyprlaptop.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/dsrosen6/hyprlaptop/internal/hypr"
)

const (
	cfgDirName  = "hyprlaptop"
	cfgFileName = "config.json"
)

type Config struct {
	path             string
	LaptopMonitor    hypr.Monitor            `json:"laptop_monitor"`
	ExternalMonitors map[string]hypr.Monitor `json:"external_monitors"`
}

func InitConfig(path string) (*Config, error) {
	uc, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("getting user config directory path: %w", err)
	}
	defPath := filepath.Join(uc, cfgDirName, cfgFileName)

	if path == "" {
		path = defPath
	}

	return readConfig(path)
}

func defaultCfg(path string) *Config {
	return &Config{
		path:             path,
		LaptopMonitor:    hypr.Monitor{},
		ExternalMonitors: map[string]hypr.Monitor{},
	}
}

func readConfig(path string) (*Config, error) {
	cfg := &Config{}
	if _, err := os.Stat(path); err != nil {
		slog.Info("no config file found; creating default")
		cfg = defaultCfg(path)

		if err := cfg.Write(); err != nil {
			return nil, fmt.Errorf("creating default config file: %w", err)
		}
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if err := json.Unmarshal(file, cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling json: %w", err)
	}

	cfg.path = path
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.LaptopMonitor.Name == "" {
		return errors.New("laptop monitor name not set")
	}

	return nil
}

func (c *Config) Write() error {
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("checking and/or creating config directory: %w", err)
	}

	str, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling json: %w", err)
	}

	if err := os.WriteFile(c.path, str, 0o644); err != nil {
		return fmt.Errorf("writing to file: %w", err)
	}

	return nil
}
