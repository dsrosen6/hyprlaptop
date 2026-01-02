package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

const (
	cfgDirName  = "hyprlaptop"
	cfgFileName = "config.json"
)

type (
	config struct {
		path     string
		Monitors map[string]monitorConfig
		Profiles []Profile `json:"profiles"`
	}

	monitorConfig struct{}
)

func defaultCfg(path string) *config {
	return &config{
		path:     path,
		Profiles: []Profile{},
	}
}

func InitConfig(path string) (*config, error) {
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

func (c *config) Path() string {
	return c.path
}

func (c *config) Reload(maxRetries int) error {
	u, err := readConfigWithRetry(c.path, maxRetries)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	if u.Profiles != nil {
		c.Profiles = u.Profiles
	}
	return nil
}

func (c *config) Write() error {
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

func readConfig(path string, createDefault bool) (*config, error) {
	cfg := &config{}
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

	cfg.path = path
	return cfg, nil
}

func readConfigWithRetry(path string, maxRetries int) (*config, error) {
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
