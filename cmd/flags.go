package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultDir  = "hypr"
	defaultFile = "hyprlaptop.json"
)

var cfgFile string

// parseFlags parses command line flags (currently just config path override)
func parseFlags() error {
	flag.StringVar(&cfgFile, "c", "", "specify a config file")
	flag.Parse()
	if cfgFile == "" {
		cd, err := os.UserConfigDir()
		if err != nil {
			return fmt.Errorf("getting user config directory: %w", err)
		}
		cfgFile = filepath.Join(cd, defaultDir, defaultFile)
	}

	return nil
}
