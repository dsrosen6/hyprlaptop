// Package config handles all configuration logic for hyprlaptop.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/dsrosen6/hyprlaptop/internal/hypr"
)

const (
	cfgDirName  = "hyprlaptop"
	cfgFileName = "config.json"
)

type Config struct {
	Path             string
	LaptopDisplay    hypr.Monitor            `json:"laptop_display"`
	ExternalDisplays map[string]hypr.Monitor `json:"external_displays"`
}

func defaultCfg(path string) *Config {
	return &Config{
		Path:             path,
		LaptopDisplay:    hypr.Monitor{},
		ExternalDisplays: map[string]hypr.Monitor{},
	}
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

	return readConfig(path, true)
}

func (c *Config) Validate() error {
	if c.LaptopDisplay.Name == "" {
		return errors.New("laptop display name not set")
	}

	return nil
}

func (c *Config) Reload(maxRetries int) error {
	u, err := readConfigWithRetry(c.Path, maxRetries)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	c.LaptopDisplay = u.LaptopDisplay
	c.ExternalDisplays = u.ExternalDisplays
	return nil
}

func (c *Config) Write() error {
	dir := filepath.Dir(c.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("checking and/or creating config directory: %w", err)
	}

	str, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling json: %w", err)
	}

	if err := os.WriteFile(c.Path, str, 0o644); err != nil {
		return fmt.Errorf("writing to file: %w", err)
	}

	return nil
}

func readConfig(path string, createDefault bool) (*Config, error) {
	cfg := &Config{}
	if _, err := os.Stat(path); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("stat config file: %w", err)
		}

		if !createDefault {
			return nil, os.ErrNotExist
		}

		slog.Info("no config file found; creating default", "path", path)
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

	cfg.Path = path
	return cfg, nil
}

func readConfigWithRetry(path string, maxRetries int) (*Config, error) {
	var lastErr error

	for i := range maxRetries {
		cfg, err := readConfig(path, false)
		if err == nil {
			return cfg, nil
		}

		lastErr = err
		time.Sleep(time.Duration(50*(i+1)) * time.Millisecond)
	}

	return nil, fmt.Errorf("config read failed after %d retries: %w", maxRetries, lastErr)
}
