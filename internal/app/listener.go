package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dsrosen6/hyprlaptop/internal/listener"
)

func (a *App) Listen(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	events := make(chan listener.Event, 16)
	errc := make(chan error, 1)

	go func() {
		if err := listener.ListenForEvents(ctx, a.Cfg.Path, events); err != nil {
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
			case listener.DisplayAddEvent, listener.DisplayRemoveEvent, listener.DisplayUnknownEvent:
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
