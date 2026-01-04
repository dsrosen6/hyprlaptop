package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"time"

	"github.com/godbus/dbus/v5"
)

type app struct {
	hctl          *hyprClient
	cfg           *config
	listener      *listener
	currentState  *state
	updating      bool
	lastUpdateEnd time.Time
}

func newApp(cfg *config, hc *hyprClient, l *listener, s *state) *app {
	return &app{
		hctl:         hc,
		cfg:          cfg,
		listener:     l,
		currentState: s,
	}
}

func run() error {
	if os.Getenv("DEBUG") == "true" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

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

	s, err := getInitialState(context.Background(), dc, h)
	if err != nil {
		return fmt.Errorf("getting initial state: %w", err)
	}

	app := newApp(cfg, h, l, s)
	app.validateAllProfiles()

	// initial updater run before starting listener
	_ = app.runUpdater()

	return app.listen(context.Background())
}

// listen starts hyprlaptop's listener, which handles hyprctl display add/remove events
// and events from the hyprlaptop CLI.
func (a *app) listen(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	events := make(chan listenerEvent, 16)
	errc := make(chan error, 1)

	go func() {
		slog.Info("listening for updates")
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

			slog.Debug("received event from listener", "type", ev.Type, "details", ev.Details)
			switch ev.Type {
			case displayInitialEvent, displayAddEvent,
				displayRemoveEvent, displayUnknownEvent:
				m, err := a.hctl.listMonitors()
				if err != nil {
					slog.Error("listing current monitors", "error", err)
					continue
				}
				if !reflect.DeepEqual(a.currentState.Monitors, m) {
					a.currentState.Monitors = m
					slog.Debug("monitors state updated", "state", a.currentState.Monitors)
				}

			case lidSwitchEvent:
				a.currentState.LidState = parseLidState(ev.Details)
				slog.Debug("lid state updated", "state", a.currentState.LidState)

			case powerChangedEvent:
				a.currentState.PowerState = parsePowerState(ev.Details)
				slog.Debug("power state updated", "state", a.currentState.PowerState)

			case configUpdatedEvent:
				// Update config values
				err := a.cfg.reload(5)
				if err != nil {
					slog.Error("reloading config", "error", err)
					continue
				}
				slog.Info("profiles reloaded", "count", len(a.cfg.Profiles))
				a.validateAllProfiles()
			}

			if !a.currentState.ready() {
				slog.Debug("not ready; awaiting initial values")
				continue
			}

			if a.updating || time.Since(a.lastUpdateEnd) < 500*time.Millisecond {
				slog.Debug("skipping: in cooldown")
				continue
			}

			if err := a.runUpdater(); err != nil {
				slog.Error("running updater", "error", err)
			}

		case err := <-errc:
			return fmt.Errorf("listener failed: %w", err)

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func getInitialState(ctx context.Context, dc *dbus.Conn, hc *hyprClient) (*state, error) {
	// TODO: theres gotta be a better way
	ls, err := newLidHandler(dc).getCurrentState(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting lid status: %w", err)
	}

	ps, err := newPowerHandler(dc).getCurrentState(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting power status: %w", err)
	}

	m, err := hc.listMonitors()
	if err != nil {
		return nil, fmt.Errorf("listing monitors: %w", err)
	}

	return &state{
		LidState:   ls,
		PowerState: ps,
		Monitors:   m,
	}, nil
}
