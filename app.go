package main

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/godbus/dbus/v5"
)

type app struct {
	hctl     *hyprClient
	cfg      *config
	listener *listener
	monitors monitorConfigMap
	profiles []*profile
	state    *state
}

func newApp(cfg *config, hc *hyprClient, l *listener) *app {
	return &app{
		hctl:     hc,
		cfg:      cfg,
		listener: l,
		monitors: cfg.Monitors,
		profiles: cfg.Profiles,
		state:    &state{},
	}
}

func run() error {
	cfg, err := initConfig("")
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
	app.validateAllProfiles()

	return app.listen(context.Background())
}

func (a *app) RunUpdater() {
	matched := a.getMatchingProfile()
	if matched == nil {
		slog.Info("no match found")
		return
	}
	slog.Info("found profile match", "profile", matched.Name)
}

// listen starts hyprlaptop's listener, which handles hyprctl display add/remove events
// and events from the hyprlaptop CLI.
func (a *app) listen(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	events := make(chan listenerEvent, 16)
	errc := make(chan error, 1)

	go func() {
		if err := a.listener.listen(ctx, events); err != nil {
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
				m, err := a.hctl.listMonitors()
				if err != nil {
					slog.Error("listing current monitors", "error", err)
					continue
				}
				if !reflect.DeepEqual(a.state.Monitors, m) {
					a.state.Monitors = m
					slog.Info("monitors state updated", "state", a.state.Monitors)
				}

			case lidSwitchEvent:
				a.state.LidState = parseLidState(ev.Details)
				slog.Info("lid state updated", "state", a.state.LidState)

			case powerChangedEvent:
				a.state.PowerState = parsePowerState(ev.Details)
				slog.Info("power state updated", "state", a.state.PowerState)

			case configUpdatedEvent:
				// Update config values
				err := a.cfg.reload(5)
				if err != nil {
					slog.Error("reloading config", "error", err)
					continue
				}
				a.monitors = a.cfg.Monitors
				a.profiles = a.cfg.Profiles
				slog.Info("profiles reloaded", "count", len(a.profiles))
				a.validateAllProfiles()
			}

			if !a.state.ready() {
				continue
			}

			a.RunUpdater()
		case err := <-errc:
			return fmt.Errorf("listener failed: %w", err)

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
