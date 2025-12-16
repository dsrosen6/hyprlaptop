package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dsrosen6/hyprlaptop/internal/listener"
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
			// All of these do the same thing. They are separate events for logging and for potential
			// logic if they need to do different things in the future.
			case listener.DisplayAddEvent, listener.DisplayRemoveEvent, listener.LidSwitchEvent,
				listener.IdleWakeEvent, listener.DisplayUnknownEvent:
				if err := a.Run(); err != nil {
					slog.Error("running display updater", "error", err)
				}

			case listener.ConfigUpdatedEvent:
				// Update config values
				err := a.Cfg.Reload(5)
				if err != nil {
					slog.Error("reloading config", "error", err)
				} else {
					// Run displayer updater in case changes are needed from new config values
					if err := a.Run(); err != nil {
						slog.Error("running display updater (config change)", "error", err)
					}
				}
			}

		case err := <-errc:
			return fmt.Errorf("listener failed: %w", err)

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
