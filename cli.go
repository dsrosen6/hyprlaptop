package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
)

const (
	defaultDir  = "hyprlaptop"
	defaultFile = "config.json"
)

var cfgFile string

type options struct {
	configFile string
}

func parseFlags() (*options, error) {
	flag.StringVar(&cfgFile, "c", "", "specify a config file")

	if cfgFile == "" {
		cd, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("getting user config directory: %w", err)
		}
		cfgFile = filepath.Join(cd, defaultDir, defaultFile)
	}
	return &options{
		configFile: cfgFile,
	}, nil
}

func parseArgs(c *client) error {
	args := os.Args
	if len(args) > 1 {
		switch args[1] {
		case "select-monitor":
			m, err := pickMonitor(c)
			if err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					return nil
				}
				return err
			}
			fmt.Println(m.Name)
		}
	}
	return nil
}
