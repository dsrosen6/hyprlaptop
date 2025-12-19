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
	version = "0.1.1"
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
		return handleRefresh()
	}

	switch args[0] {
	case "version":
		fmt.Println(version)
		return nil
	case "save-displays", "sd":
		return handleSaveDisplays(args)
	case "lid", "lid-switch":
		return handleLidSwitch()
	case "wake":
		return handleWake()
	case "listen":
		return handleListen(ctx)
	default:
		return errors.New("invalid command")
	}
}

// handleRefresh runs a regular refresh of hyprlaptop if no subcommands
// are passed. It is a manual or catchall run.
func handleRefresh() error {
	if err := a.Run(); err != nil {
		return fmt.Errorf("refreshing: %w", err)
	}

	return nil
}

// handleSaveDisplays saves the current arrangement of displays into the config,
// essentially freezing the setup state for future runs. This is a way around
// manually inputting your config.
func handleSaveDisplays(args []string) error {
	expectedArgs := 1
	gotArgs := len(args) - 1
	if gotArgs != expectedArgs {
		return fmt.Errorf("expected %d arguments, got %d", expectedArgs, gotArgs)
	}

	if err := saveDiplaysCmd.Parse(args[1:]); err != nil {
		return fmt.Errorf("parsing arguments: %w", err)
	}

	if err := a.SaveCurrentDisplays(*mtrName); err != nil {
		return fmt.Errorf("setting laptop display: %w", err)
	}

	fmt.Printf("Laptop display '%s' saved to config.\n", a.Cfg.LaptopDisplay.Name)
	externals := a.Cfg.ExternalDisplays
	switch len(externals) {
	case 0:
		fmt.Println("No external display detected.")
	default:
		fmt.Println("Saved external display(s):")
		for _, e := range externals {
			fmt.Printf("	%s\n", e.Name)
		}
	}

	return nil
}

// handleLidSwitch handles the lid switch; meant to be wired up to binds in hyprland.
func handleLidSwitch() error {
	if err := app.SendLidCommand(); err != nil {
		return fmt.Errorf("sending lid switch command: %w", err)
	}

	return nil
}

// handleWake handles wake from suspend; meant to be put into "after_sleep_cmd" in hypridle.
// This handles situations where, perhaps, you shut your laptop (suspending it) and then
// it is plugged into a dock. Otherwise, it would wake with the laptop display on while it is shut.
func handleWake() error {
	if err := app.SendWakeCommand(); err != nil {
		return fmt.Errorf("sending wake command: %w", err)
	}

	return nil
}

// handleListen is the entry point to the listener; meant to be run as a systemd user unit
// or as an exec-once in hyprland, depending on if you're using UWSM.
func handleListen(ctx context.Context) error {
	slog.Info("initializing socket connection")
	slog.Info("listening for hyprland events")
	if err := a.Listen(ctx); err != nil {
		return err
	}

	return nil
}
