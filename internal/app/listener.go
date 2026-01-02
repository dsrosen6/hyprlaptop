package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dsrosen6/hyprlaptop/internal/listener"
	"github.com/dsrosen6/hyprlaptop/internal/power"
)

// Listen starts hyprlaptop's listener, which handles hyprctl display add/remove events
// and events from the hyprlaptop CLI.
func (a *App) Listen(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	events := make(chan listener.Event, 16)
	errc := make(chan error, 1)

	go func() {
		if err := listener.ListenForEvents(ctx, a.Cfg.Path(), events); err != nil {
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
			case listener.LidSwitchEvent:
				a.State.LidState = power.ParseLidState(ev.Details)
				slog.Info("lid state updated", "state", a.State.LidState.String())

			case listener.PowerChangedEvent:
				a.State.PowerState = power.ParsePowerState(ev.Details)
				slog.Info("power state updated", "state", a.State.PowerState.String())

			case listener.ConfigUpdatedEvent:
				// Update config values
				err := a.Cfg.Reload(5)
				if err != nil {
					slog.Error("reloading config", "error", err)
					continue
				}
			}

			if !a.PowerStatesReady() {
				slog.Debug("power states not ready; not running updater")
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
