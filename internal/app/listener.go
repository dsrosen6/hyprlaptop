package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dsrosen6/hyprlaptop/internal/hypr"
)

func (a *App) Listen(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	events := make(chan hypr.Event, 16)
	errc := make(chan error, 1)

	sock, err := hypr.NewSocketConn()
	if err != nil {
		return fmt.Errorf("initializing socket connection: %w", err)
	}

	defer func() {
		if err := sock.Close(); err != nil {
			slog.Error("closing socket connection", "error", err)
		}
	}()

	go func() {
		if err := sock.ListenForEvents(ctx, events); err != nil {
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
			slog.Info("received event from listener", "name", ev.Name, "payload", ev.Payload)
			if err := a.Run(); err != nil {
				slog.Error("error running display updater: %w", err)
			}

		case err := <-errc:
			return fmt.Errorf("listener failed: %w", err)

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
