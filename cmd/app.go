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

var (
	a              *app.App
	saveDiplaysCmd = flag.NewFlagSet("save-displays", flag.ExitOnError)
	mtrName        = saveDiplaysCmd.String("laptop", "", "name of laptop display")
)

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

func handleCommands(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return handleRunOnce()
	}

	switch args[0] {
	case "save-displays", "sd":
		return handleSaveDisplays(args)
	case "listen":
		return handleListen(ctx)
	default:
		return errors.New("invalid command")
	}
}

func handleRunOnce() error {
	if err := a.Run(); err != nil {
		return fmt.Errorf("running once: %w", err)
	}

	return nil
}

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

func handleListen(ctx context.Context) error {
	slog.Info("initializing socket connection")
	slog.Info("listening for hyprland events")
	if err := a.Listen(ctx); err != nil {
		return err
	}

	return nil
}
