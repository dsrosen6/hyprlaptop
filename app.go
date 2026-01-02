package main

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/godbus/dbus/v5"
)

type App struct {
	Hctl     *hyprClient
	Cfg      *config
	Listener *listener
	Profiles []Profile
	State    *State
}

func newApp(cfg *config, hc *hyprClient, l *listener) *App {
	return &App{
		Hctl:     hc,
		Cfg:      cfg,
		Listener: l,
		State:    &State{},
	}
}

func Run() error {
	cfg, err := InitConfig("")
	if err != nil {
		return fmt.Errorf("initializing config: %w", err)
	}

	h, err := newHyprctlClient()
	if err != nil {
		return fmt.Errorf("creating hyprctl client: %w", err)
	}

	var (
		hs *hyprSocketConn
		dc *dbus.Conn
	)

	defer func() {
		if hs != nil {
			if err := hs.Close(); err != nil {
				slog.Error("closing hypr socket connection", "error", err)
			}
		}

		if dc != nil {
			if err := dc.Close(); err != nil {
				slog.Error("closing dbus connection", "error", err)
			}
		}
	}()

	hs, err = newHyprSocketConn()
	if err != nil {
		return fmt.Errorf("creating hyprland socket connection: %w", err)
	}

	dc, err = dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("creating dbus connection: %w", err)
	}

	l, err := newListener(hs, dc, cfg.path)
	if err != nil {
		return fmt.Errorf("creating listener: %w", err)
	}

	app := newApp(cfg, h, l)
	return app.Listen(context.Background())
}

func (a *App) RunUpdater() error {
	return nil
}

// Listen starts hyprlaptop's listener, which handles hyprctl display add/remove events
// and events from the hyprlaptop CLI.
func (a *App) Listen(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	events := make(chan listenerEvent, 16)
	errc := make(chan error, 1)

	go func() {
		if err := a.Listener.listen(ctx, events); err != nil {
			errc <- err
			cancel()
		}
	}()

	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return nil // normal shutdown
			}

			slog.Info("received event from listener", "type", ev.Type, "details", ev.Details)
			switch ev.Type {
			case displayInitialEvent, displayAddEvent,
				displayRemoveEvent, displayUnknownEvent:
				m, err := a.Hctl.listMonitors()
				if err != nil {
					slog.Error("listing current monitors", "error", err)
					continue
				}
				if !reflect.DeepEqual(a.State.Monitors, m) {
					a.State.Monitors = m
					slog.Info("monitors state updated", "state", a.State.Monitors)
				}

			case lidSwitchEvent:
				a.State.LidState = parseLidState(ev.Details)
				slog.Info("lid state updated", "state", a.State.LidState.string())

			case powerChangedEvent:
				a.State.PowerState = parsePowerState(ev.Details)
				slog.Info("power state updated", "state", a.State.PowerState.string())

			case configUpdatedEvent:
				// Update config values
				err := a.Cfg.Reload(5)
				if err != nil {
					slog.Error("reloading config", "error", err)
					continue
				}
				a.Profiles = a.Cfg.Profiles
				slog.Info("profiles reloaded", "count", len(a.Profiles))
			}

			if !a.State.Ready() {
				continue
			}

			if err := a.RunUpdater(); err != nil {
				slog.Error("running updater", "error", err)
			}

		case err := <-errc:
			return fmt.Errorf("listener failed: %w", err)

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
