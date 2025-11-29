package main

import (
	"fmt"
	"log/slog"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func run() error {
	opts, err := parseFlags()
	if err != nil {
		return fmt.Errorf("parsing cli: %w", err)
	}

	c, err := readConfig(opts.configFile)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	slog.Info("config loaded", "primary_monitor", c.LaptopMonitor.Name)

	if os.Getenv("DEBUG") == "true" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	hc, err := newClient()
	if err != nil {
		return fmt.Errorf("creating hypr client: %w", err)
	}

	if err := parseArgs(hc); err != nil {
		return err
	}
	// conn, err := NewConn()
	// if err != nil {
	// 	return err
	// }

	// if err := conn.Listen(); err != nil {
	// 	return err
	// }

	return nil
}
