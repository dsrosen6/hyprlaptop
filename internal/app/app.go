package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dsrosen6/hyprdocked/internal/hypr"
	"github.com/dsrosen6/hyprdocked/internal/power"
	"github.com/godbus/dbus/v5"
	"github.com/spf13/viper"
)

type App struct {
	Config            Config
	hctl              *hypr.Client
	listener          *listener
	updating          bool
	configReloadTimer *time.Timer
	*state
}

type RunParams struct {
	LaptopMonitorName string
	SuspendOnIdle     bool
	SuspendOnClosed   bool
}

func newApp(cfg Config, hc *hypr.Client, l *listener, s *state) *App {
	return &App{
		Config:   cfg,
		hctl:     hc,
		listener: l,
		state:    s,
	}
}

func RunListener(c Config) error {
	if err := hypr.CheckEnvs(); err != nil {
		return fmt.Errorf("checking hyprland environment: %w", err)
	}
	if c.Laptop == "" {
		return errors.New("laptop monitor name cannot be empty")
	}

	hyprClient, err := hypr.NewClient()
	if err != nil {
		return fmt.Errorf("creating hyprctl client: %w", err)
	}

	// Run an initial reload in case laptop display is already disabled. Assuming the laptop
	// display is correctly set to initially enable in the hyprland config, this will re-enable
	// it so hyprdocked can properly identify it.
	slog.Info("running hyprctl reload")
	if err := hyprClient.Reload(); err != nil {
		return fmt.Errorf("running hyprctl reload: %w", err)
	}

	var (
		hyprSock *hypr.SocketConn
		dbusConn *dbus.Conn
	)

	defer func() {
		if hyprSock != nil {
			if err := hyprSock.Close(); err != nil {
				slog.Error("closing hypr socket connection", "error", err)
			}
		}

		if dbusConn != nil {
			if err := dbusConn.Close(); err != nil {
				slog.Error("closing dbus connection", "error", err)
			}
		}
	}()
	hyprSock, err = hypr.NewSocketConn()
	if err != nil {
		return fmt.Errorf("creating hyprland socket connection: %w", err)
	}

	dbusConn, err = dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("creating dbus connection: %w", err)
	}

	lh := power.NewLidHandler(dbusConn)
	lp := listenerParams{
		hyprSockConn: hyprSock,
		lidHandler:   lh,
		dbusConn:     dbusConn,
	}

	l, err := newListener(lp)
	if err != nil {
		return fmt.Errorf("creating listener: %w", err)
	}

	sp := initialStateParams{
		laptopMonitorName: c.Laptop,
		hyprClient:        hyprClient,
		lidHandler:        lh,
	}

	s, err := getInitialState(context.Background(), sp)
	if err != nil {
		return fmt.Errorf("getting initial state: %w", err)
	}

	a := newApp(c, hyprClient, l, s)
	slog.Info("app initialized",
		"laptop_monitor_name", a.laptopDisplay.Name,
		"status", a.statusString(),
		"suspend_idle", a.Config.SuspendIdle,
		"suspend_closed", a.Config.SuspendClosed,
	)

	// initial updater run before starting listener
	changed, _ := a.runUpdater()
	a.runPostHooks(changed)

	viper.OnConfigChange(a.onConfigChange)
	viper.WatchConfig()

	return a.listenAndHandle(context.Background())
}
