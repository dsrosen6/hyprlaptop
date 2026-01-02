package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/dsrosen6/hyprlaptop/internal/app"
	"github.com/dsrosen6/hyprlaptop/internal/config"
	"github.com/dsrosen6/hyprlaptop/internal/hypr"
)

const (
	version = "0.1.2"
)

var (
	a              *app.App
	saveDiplaysCmd = flag.NewFlagSet("save-displays", flag.ExitOnError)
	mtrName        = saveDiplaysCmd.String("laptop", "", "name of laptop display")
)

// Run is the primary entry point of hyprlaptop. It is used both to launch the listener
// and to handle CLI commands.
func Run() error {
	ctx := context.Background()
	if err := parseFlags(); err != nil {
		return fmt.Errorf("parsing cli flags: %w", err)
	}

	if os.Getenv("DEBUG") == "true" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	cfg, err := config.InitConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}
	slog.Debug("initiated config", "path", cfg.Path)

	hc, err := hypr.NewHyprctlClient()
	if err != nil {
		return fmt.Errorf("creating hyprctl client: %w", err)
	}

	a = app.NewApp(cfg, hc)
	if err := handleCommands(ctx, os.Args[1:]); err != nil {
		return err
	}

	return nil
}

// handleCommands parses and handles CLI options.
func handleCommands(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return handleListen(ctx)
	}

	switch args[0] {
	case "version":
		fmt.Println(version)
		return nil
	default:
		return errors.New("invalid command")
	}
}

// handleListen is the entry point to the listener; meant to be run as a systemd user unit
// or as an exec-once in hyprland, depending on if you're using UWSM.
func handleListen(ctx context.Context) error {
	if err := a.Listen(ctx); err != nil {
		return err
	}

	return nil
}
