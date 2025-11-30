package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
)

type app struct {
	hc      *hyprctlClient
	cfg     *config
	cfgPath string
}

var (
	monitorsCmd = flag.NewFlagSet("save-monitors", flag.ExitOnError)
	laptopMtr   = monitorsCmd.String("laptop", "", "name of laptop monitor")
)

func newApp() (*app, error) {
	if os.Getenv("DEBUG") == "true" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	opts, err := parseFlags()
	if err != nil {
		return nil, fmt.Errorf("parsing cli flags: %w", err)
	}

	cfg, err := readConfig(opts.configFile)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	hc, err := newHctlClient()
	if err != nil {
		return nil, fmt.Errorf("creating hyprctl client: %w", err)
	}

	return &app{
		hc:      hc,
		cfg:     cfg,
		cfgPath: opts.configFile,
	}, nil
}

func (a *app) run() error {
	args := os.Args
	if len(args) < 2 {
		return errors.New("no subcommand provided")
	}

	switch args[1] {
	case "save-monitors", "sm":
		return a.handleSaveMonitors(args[1:])
	case "listen":
		return a.handleListen()
	default:
		return errors.New("invalid command")
	}
}

func (a *app) handleSaveMonitors(args []string) error {
	expectedArgs := 1
	gotArgs := len(args) - 1
	if gotArgs != expectedArgs {
		return fmt.Errorf("expected %d arguments, got %d", expectedArgs, gotArgs)
	}

	if err := monitorsCmd.Parse(args[1:]); err != nil {
		return fmt.Errorf("parsing arguments: %w", err)
	}

	if err := a.saveCurrentMonitors(*laptopMtr); err != nil {
		return fmt.Errorf("setting laptop monitor: %w", err)
	}

	fmt.Printf("Laptop monitor '%s' saved to config.\n", a.cfg.LaptopMonitor.Name)
	externals := a.cfg.ExternalMonitors
	switch len(externals) {
	case 0:
		fmt.Println("No external monitors detected.")
	default:
		fmt.Println("Saved external monitor(s):")
		for _, e := range externals {
			fmt.Printf("	%s\n", e.Name)
		}
	}

	return nil
}

func (a *app) handleListen() error {
	sc, err := newSocketConn()
	if err != nil {
		return fmt.Errorf("initializing socket connection: %w", err)
	}
	defer func() {
		if err := sc.Close(); err != nil {
			slog.Error("closing socket connection", "error", err)
		}
	}()

	slog.Info("listening for hyprland events")
	if err := a.listen(sc); err != nil {
		return err
	}

	return nil
}
